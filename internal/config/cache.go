package config

import (
	"errors"
	"os"
	"time"
)

const (
	cacheDir = "/var/cache/xray-vpn"
	cacheTTL = 1 * 12 * time.Hour
)

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

func readCache(path string) cacheResult {
	info, statErr := os.Stat(path)
	if statErr != nil {
		if errors.Is(statErr, os.ErrNotExist) {
			return cacheResult{State: cacheMissing}
		}
		return cacheResult{State: cacheError, Err: statErr}
	}

	data, readErr := os.ReadFile(path)
	if readErr != nil {
		if errors.Is(readErr, os.ErrNotExist) {
			return cacheResult{State: cacheMissing}
		}
		return cacheResult{State: cacheError, Err: readErr}
	}

	if time.Since(info.ModTime()) > cacheTTL {
		return cacheResult{State: cacheStale, Data: data}
	}
	return cacheResult{State: cacheFresh, Data: data}
}
