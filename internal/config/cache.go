package config

import (
	"errors"
	"os"
	"time"

	"github.com/xtls/xray-core/common/platform"
)

const cacheTTL = 2 * time.Hour

type cacheState int

const (
	cacheFresh   cacheState = iota // exists, readable, within TTL
	cacheStale                     // exists, readable, TTL expired
	cacheMissing                   // file does not exist
	cacheError                     // I/O error (stat or read)
)

type cacheResult struct {
	State cacheState
	Data  []byte
	Err   error
}

func readCache(name string) *cacheResult {
	path := platform.GetAssetLocation(name)
	info, statErr := os.Stat(path)
	if statErr != nil {
		if errors.Is(statErr, os.ErrNotExist) {
			return &cacheResult{State: cacheMissing}
		}
		return &cacheResult{State: cacheError, Err: statErr}
	}

	data, readErr := os.ReadFile(path)
	if readErr != nil {
		if errors.Is(readErr, os.ErrNotExist) {
			return &cacheResult{State: cacheMissing}
		}
		return &cacheResult{State: cacheError, Err: readErr}
	}

	if time.Since(info.ModTime()) > cacheTTL {
		return &cacheResult{State: cacheStale, Data: data}
	}
	return &cacheResult{State: cacheFresh, Data: data}
}

func writeCache(name string, data []byte) error {
	return writeCacheFrom(name, func(f *os.File) error {
		_, err := f.Write(data)
		return err
	})
}

func writeCacheFrom(name string, write func(*os.File) error) error {
	dest := platform.GetAssetLocation(name)
	tmp := dest + ".tmp"

	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0660)
	if err != nil {
		return err
	}
	defer os.Remove(tmp)

	err = write(f)
	f.Close()
	if err != nil {
		return err
	}

	return os.Rename(tmp, dest)
}
