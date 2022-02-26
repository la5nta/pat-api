package main

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
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
	Generated  time.Time `json:"_generated"`
}

func (f FormsInfo) String() string {
	return fmt.Sprintf("version: '%s', url: '%s'", f.Version, f.ArchiveURL)
}

const FormsInfoURL = "https://www.winlink.org/content/all_standard_templates_folders_one_zip_self_extracting_winlink_express_ver_12142016"
const PatFormsAPIPath = "https://api.getpat.io/v1/forms/standard-templates/"

var client = &http.Client{Timeout: 10 * time.Second}

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
