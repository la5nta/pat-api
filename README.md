# pat-api

[![Update Forms Version](https://github.com/la5nta/pat-api/actions/workflows/go.yml/badge.svg)](https://github.com/la5nta/pat-api/actions/workflows/go.yml)

APIs supporting [Pat](https://github.com/la5nta/pat). The [`ghpages` branch](https://github.com/la5nta/pat-api/tree/ghpages) contains files 
which are updated by the scripts in the `main` branch.

## Updating Standard Forms

Run `./update-forms.sh` locally. The script creates or reuses a `ghpages`
worktree at `./ghpages`, synchronizes it with `origin/ghpages`, downloads and
validates the latest forms, and commits any changes. The only manual step is to
push the resulting commit:

```sh
git -C ghpages push origin ghpages
```

## Routes

```
- GET https://api.getpat.io/v1/forms/standard-templates/latest
- GET https://api.getpat.io/v1/releases/latest
```
