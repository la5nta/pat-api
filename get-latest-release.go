package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

type Release struct {
	Version    string `json:"version"`
	ReleaseURL string `json:"release_url"`

	// To keep the gh action from being disabled due to repo inactivity
	GhKeepAlive KeepAliveToken `json:"_gh_keepalive"`
}

func main() {
	release, err := getLatestRelease("la5nta/pat")
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Found version %s", release.Version)
	_ = json.NewEncoder(os.Stdout).Encode(release)
}

func getLatestRelease(repo string) (Release, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases?per_page=1", repo)

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return Release{}, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return Release{}, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return Release{}, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var releases []struct {
		TagName string `json:"tag_name"`
		HTMLURL string `json:"html_url"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return Release{}, err
	}

	if len(releases) == 0 {
		return Release{}, fmt.Errorf("no releases found")
	}

	// Strip 'v' prefix if present
	version := strings.TrimPrefix(releases[0].TagName, "v")
	return Release{
		Version:    version,
		ReleaseURL: releases[0].HTMLURL,
	}, nil
}
