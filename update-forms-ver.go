package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"regexp"
	"time"
)

type FormsInfo struct {
	Version    string    `json:"version"`
	ArchiveURL string    `json:"archive_url"`
	Generated  time.Time `json:"-"`
}

func (f FormsInfo) String() string {
	return fmt.Sprintf("version: '%s', url: '%s'", f.Version, f.ArchiveURL)
}

const (
	FormsInfoURL    = "https://www.winlink.org/content/how_manually_update_standard_templates"
	PatFormsAPIPath = "https://api.getpat.io/v1/forms/standard-templates/"
)

var client = &http.Client{Timeout: 30 * time.Second}

func main() {
	latest, err := getLatestFormsInfo()
	if err != nil {
		log.Fatalf("could not get latest forms info: %v", err)
	}
	log.Printf("Found %s.", latest)
	filename := fmt.Sprintf("Standard_Forms_%s.zip", latest.Version)
	if err := downloadZipURL(latest.ArchiveURL, filename); err != nil {
		log.Fatalf("could not download archive url: %v", err)
	}
	latest.ArchiveURL = fmt.Sprintf("%s%s", PatFormsAPIPath, filename)
	json.NewEncoder(os.Stdout).Encode(latest)
}

func downloadZipURL(url string, filename string) error {
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusFound, http.StatusOK:
		break
	default:
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	b, err := readAndCheckZip(resp.Body)
	if err != nil {
		return err
	}

	out, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, bytes.NewReader(b))
	return err
}

func readAndCheckZip(rc io.ReadCloser) ([]byte, error) {
	// https://stackoverflow.com/a/50539327/587091
	body, err := ioutil.ReadAll(rc)
	if err != nil {
		return nil, err
	}

	zipReader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return nil, err
	}

	// Read all the files from zip archive
	for _, zipFile := range zipReader.File {
		if err = readZipFile(zipFile); err != nil {
			return nil, err
		}
	}

	return body, nil
}

func readZipFile(zf *zip.File) error {
	f, err := zf.Open()
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(io.Discard, f)
	return err
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
	hrefRe := regexp.MustCompile(`<a href="(https://.+)">\s*Standard_Forms - Version (\d+\.\d+\.\d+(\.\d+)?)\s*</a>`)
	hrefMatches := hrefRe.FindStringSubmatch(bodyString)
	if len(hrefMatches) < 3 {
		return nil, errors.New("can't scrape the version info page, HTML structure may have changed")
	}
	return &FormsInfo{
		Version:    hrefMatches[2],
		ArchiveURL: html.UnescapeString(hrefMatches[1]),
		Generated:  time.Now().UTC(),
	}, nil
}
