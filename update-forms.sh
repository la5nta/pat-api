#!/bin/sh

set -eu

root=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
worktree="$root/ghpages"
tmp=$(mktemp -d "${TMPDIR:-/tmp}/pat-api-update.XXXXXX")
trap 'rm -rf "$tmp"' EXIT HUP INT TERM

if [ ! -e "$worktree/.git" ]; then
	if [ -e "$worktree" ]; then
		printf '%s\n' "error: $worktree exists but is not a git worktree" >&2
		exit 1
	fi
	git -C "$root" worktree prune
	git -C "$root" worktree add "$worktree" ghpages
fi

if [ "$(git -C "$worktree" branch --show-current)" != "ghpages" ]; then
	printf '%s\n' "error: $worktree is not on the ghpages branch" >&2
	exit 1
fi

if [ -n "$(git -C "$worktree" status --short)" ]; then
	printf '%s\n' "error: $worktree has uncommitted changes" >&2
	exit 1
fi

git -C "$worktree" fetch origin ghpages
git -C "$worktree" rebase origin/ghpages

mkdir -p "$tmp/v1/forms/standard-templates"
(
	cd "$tmp"
	go run "$root/keepalive.go" "$root/update-forms-ver.go" >v1/forms/standard-templates/latest
)

mkdir -p "$worktree/v1/forms/standard-templates"
cp "$tmp/v1/forms/standard-templates/latest" "$worktree/v1/forms/standard-templates/latest"
rm -f "$worktree"/v1/forms/standard-templates/Standard_Forms_*.zip
cp "$tmp"/Standard_Forms_*.zip "$worktree/v1/forms/standard-templates/"

if [ -z "$(git -C "$worktree" status --short)" ]; then
	printf '\n%s\n' "No API changes."
	exit 0
fi

git -C "$worktree" add -- .
git -C "$worktree" commit -m "Update form template version"

printf '\n%s\n' "Committed API changes. Push manually with:"
printf 'git -C %s push origin ghpages\n' "$worktree"
