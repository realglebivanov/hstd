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
	name string
}

const baseGeodataUrl = "https://github.com/Loyalsoldier/v2ray-rules-dat/releases/latest/download/"

var geodataFiles = []geodataFile{
	{baseGeodataUrl + "geoip.dat", "geoip.dat"},
	{baseGeodataUrl + "geosite.dat", "geosite.dat"},
}

func loadGeodata() error {
	var staleFiles []geodataFile

	for _, f := range geodataFiles {
		cr := readCache(f.name)
		switch cr.State {
		case cacheFresh:
			log.Printf("using cached %s", f.name)
		case cacheStale:
			log.Printf("using stale %s, will refresh in background", f.name)
			staleFiles = append(staleFiles, f)
		case cacheMissing:
			if err := downloadFile(f.url, f.name); err != nil {
				return fmt.Errorf("download %s: %w", f.url, err)
			}
		case cacheError:
			return fmt.Errorf("read %s: %w", f.name, cr.Err)
		default:
			return fmt.Errorf("unexpected cache state %d for %s", cr.State, f.name)
		}
	}

	if len(staleFiles) > 0 {
		go refreshStateFiles(staleFiles)
	}

	return nil
}

func refreshStateFiles(files []geodataFile) {
	for _, f := range files {
		if err := downloadFile(f.url, f.name); err != nil {
			log.Printf("background geodata refresh failed for %s: %v", f.name, err)
		}
	}
}

func RefreshGeodata() error {
	for _, f := range geodataFiles {
		if err := downloadFile(f.url, f.name); err != nil {
			return fmt.Errorf("download %s: %w", f.url, err)
		}
	}
	return nil
}

func downloadFile(url, name string) error {
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

	return writeCacheFrom(name, func(f *os.File) error {
		n, err := io.Copy(f, resp.Body)
		if err != nil {
			return err
		}
		log.Printf("wrote %s (%d bytes)", f.Name(), n)
		return nil
	})
}
