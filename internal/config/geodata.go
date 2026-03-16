package config

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

type geodataFile struct {
	url  string
	path string
}

var baseGeodataUrl = "https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/"

var geodataFiles = []geodataFile{
	{baseGeodataUrl + "geoip.dat", cacheDir + "/geoip.dat"},
	{baseGeodataUrl + "geosite.dat", cacheDir + "/geosite.dat"},
}

func loadGeodata() error {
	var staleFiles []geodataFile

	for _, f := range geodataFiles {
		cr := readCache(f.path)
		switch cr.State {
		case cacheFresh:
			log.Printf("using cached %s", f.path)
		case cacheStale:
			log.Printf("using stale %s, will refresh in background", f.path)
			staleFiles = append(staleFiles, f)
		case cacheMissing:
			if err := downloadFile(f.url, f.path); err != nil {
				return fmt.Errorf("download %s: %w", f.url, err)
			}
		case cacheError:
			return fmt.Errorf("read %s: %w", f.path, cr.Err)
		default:
			return fmt.Errorf("unexpected cache state %d for %s", cr.State, f.path)
		}
	}

	if len(staleFiles) > 0 {
		go refreshStateFiles(staleFiles)
	}

	return nil
}

func refreshStateFiles(files []geodataFile) {
	for _, f := range files {
		if err := downloadFile(f.url, f.path); err != nil {
			log.Printf("background geodata refresh failed for %s: %v", f.path, err)
		}
	}
}

func RefreshGeodata() error {
	for _, f := range geodataFiles {
		if err := downloadFile(f.url, f.path); err != nil {
			return fmt.Errorf("download %s: %w", f.url, err)
		}
	}
	return nil
}

func downloadFile(url, dest string) error {
	log.Printf("downloading %s ...", url)
	client := &http.Client{Timeout: 2 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	tmp := dest + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return err
	}

	n, err := io.Copy(f, resp.Body)
	f.Close()
	if err != nil {
		os.Remove(tmp)
		return err
	}

	if err := os.Rename(tmp, dest); err != nil {
		os.Remove(tmp)
		return err
	}

	log.Printf("wrote %s (%d bytes)", dest, n)
	return nil
}
