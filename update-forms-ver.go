package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"time"
)

type FormsInfo struct {
	Version    string    `json:"version"`
	ArchiveURL string    `json:"archive_url"`
	Generated  time.Time `json:"_generated"`
}

const FormsInfoURL = "https://www.winlink.org/content/all_standard_templates_folders_one_zip_self_extracting_winlink_express_ver_12142016"

var client = &http.Client{Timeout: 10 * time.Second}

func main() {
	latest, err := getLatestFormsInfo()
	if err != nil {
		log.Fatal(err)
	}
	if err := verifyURL(latest.ArchiveURL); err != nil {
		log.Fatalf("could not verify archive url: %v", err)
	}
	json.NewEncoder(os.Stdout).Encode(latest)
}

func verifyURL(url string) error {
	resp, err := client.Head(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusFound, http.StatusOK:
		return nil
	default:
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
}

func getLatestFormsInfo() (*FormsInfo, error) {
	req, err := http.NewRequest("GET", FormsInfoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "pat-forms-scraper")
	req.Header.Set("Cache-Control", "no-cache")
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("can't read winlink forms version page: %w", err)
	}
	bodyString := string(bodyBytes)

	// Scrape for the version and download link
	versionRe := regexp.MustCompile(`Standard_Forms - Version (\d+\.\d+\.\d+(\.\d+)?)`)
	downloadRe := regexp.MustCompile(`https://1drv.ms/u/([a-zA-Z0-9-_!]+)\?e=([a-zA-Z0-9-_]+)`)
	versionMatches := versionRe.FindStringSubmatch(bodyString)
	downloadMatches := downloadRe.FindStringSubmatch(bodyString)
	if versionMatches == nil || len(versionMatches) < 2 || downloadMatches == nil || len(downloadMatches) < 3 {
		return nil, errors.New("can't scrape the version info page, HTML structure may have changed")
	}
	newestVersion := versionMatches[1]
	docID := downloadMatches[1]
	auth := downloadMatches[2]
	downloadLink := "https://api.onedrive.com/v1.0/shares/" + docID + "/root/content?e=" + auth
	return &FormsInfo{
		Version:    newestVersion,
		ArchiveURL: downloadLink,
		Generated:  time.Now().UTC(),
	}, nil
}
