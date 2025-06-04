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
	Version string `json:"version"`

	// To keep the gh action from being disabled due to repo inactivity
	GhKeepAlive KeepAliveToken `json:"_gh_keepalive"`
}

func main() {
	latestVersion, err := getLatestRelease("la5nta/pat")
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Found version %s", latestVersion)
	_ = json.NewEncoder(os.Stdout).Encode(Release{
		Version: latestVersion,
	})
}

func getLatestRelease(repo string) (string, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases?per_page=1", repo)

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var releases []struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		return "", err
	}

	if len(releases) == 0 {
		return "", fmt.Errorf("no releases found")
	}

	// Strip 'v' prefix if present
	version := strings.TrimPrefix(releases[0].TagName, "v")
	return version, nil
}
