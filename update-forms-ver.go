package main

import (
	"archive/zip"
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"regexp"
	"strings"
	"time"
	"unicode"
)

// KeepAliveToken represents a unique token per calendar month.
type KeepAliveToken struct{}

func (g KeepAliveToken) MarshalJSON() ([]byte, error) { return json.Marshal(g.String()) }

func (KeepAliveToken) String() string {
	// sha1 encoded month of year
	return fmt.Sprintf("%x", sha1.Sum([]byte{byte(time.Now().Month())}))
}

type FormsInfo struct {
	// We need this to keep the gh action from being disabled due to repo
	// inactivity if Standard Froms is not updated for a while.
	// (>= 60 days).
	GhKeepAlive KeepAliveToken `json:"_gh_keepalive"`

	Version    string `json:"version"`
	ArchiveURL string `json:"archive_url"`
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
	url, err := getLatestFormsUrl()
	if err != nil {
		log.Fatalf("could not get latest forms info: %v", err)
	}
	log.Printf("Found URL %s", url)
	latest, err := downloadZipURL(url)
	if err != nil {
		log.Fatalf("could not download archive url: %v", err)
	}
	log.Printf("Found version %s", latest.Version)
	_ = json.NewEncoder(os.Stdout).Encode(latest)
}

func downloadZipURL(zipUrl string) (*FormsInfo, error) {
	resp, err := client.Get(zipUrl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	switch resp.StatusCode {
	case http.StatusFound, http.StatusOK:
		break
	default:
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	b, version, err := readAndCheckZip(resp.Body)
	if err != nil {
		return nil, err
	}

	filename := fmt.Sprintf("Standard_Forms_%s.zip", version)
	out, err := os.Create(filename)
	if err != nil {
		return nil, err
	}
	defer out.Close()
	_, err = io.Copy(out, bytes.NewReader(b))
	return &FormsInfo{
		Version:    version,
		ArchiveURL: path.Join(PatFormsAPIPath, url.PathEscape(filename)),
	}, err
}

func readAndCheckZip(rc io.ReadCloser) ([]byte, string, error) {
	var version string
	// https://stackoverflow.com/a/50539327/587091
	body, err := io.ReadAll(rc)
	if err != nil {
		return nil, "", err
	}

	zipReader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return nil, "", err
	}

	// Read all the files from zip archive
	for _, zipFile := range zipReader.File {
		// If the file is Standard_Forms_Version.dat, read the contents
		if zipFile.Name == "Standard_Forms_Version.dat" {
			ver, err := readZipFileContents(zipFile)
			if err != nil {
				return nil, "", err
			}
			// strings.TrimSpace is not sufficient. Version 1.1.6.0 was released as `1.1.6\t.0`
			version = stripSpaces(string(ver))
		}

		// Otherwise, read the file to check for errors but discard the contents
		if err = readZipFile(zipFile); err != nil {
			return nil, "", err
		}
	}

	return body, version, nil
}

func stripSpaces(str string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			// if the character is a space, drop it
			return -1
		}
		// else keep it in the string
		return r
	}, str)
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

func readZipFileContents(zf *zip.File) ([]byte, error) {
	f, err := zf.Open()
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}

func getLatestFormsUrl() (string, error) {
	req, err := http.NewRequest("GET", FormsInfoURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "pat-forms-scraper")
	req.Header.Set("Cache-Control", "no-cache")
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("can't read winlink forms version page: %w", err)
	}
	bodyString := string(bodyBytes)

	// Scrape for the version and download link
	hrefRe := regexp.MustCompile(`<a href="(https://.+)">\s*Standard_Forms - Latest Version\s*</a>`)
	hrefMatches := hrefRe.FindStringSubmatch(bodyString)
	if len(hrefMatches) < 2 {
		return "", errors.New("can't scrape the version info page, HTML structure may have changed")
	}
	return html.UnescapeString(hrefMatches[1]), nil
}
