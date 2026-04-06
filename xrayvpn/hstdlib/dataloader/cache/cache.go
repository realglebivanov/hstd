package cache

import (
	"errors"
	"os"
	"path/filepath"
	"time"
)

const cacheTTL = 2 * time.Hour

type CacheState int

const (
	CacheFresh   CacheState = iota // exists, readable, within TTL
	CacheStale                     // exists, readable, TTL expired
	CacheMissing                   // file does not exist
	CacheError                     // I/O error (stat or read)
)

type CacheResult struct {
	State CacheState
	Data  []byte
	Err   error
}

type Cache struct {
	dir string
}

func New(dir string) *Cache {
	return &Cache{dir: dir}
}

func (c *Cache) Read(name string) *CacheResult {
	path := filepath.Join(c.dir, name)
	info, statErr := os.Stat(path)
	if statErr != nil {
		if errors.Is(statErr, os.ErrNotExist) {
			return &CacheResult{State: CacheMissing}
		}
		return &CacheResult{State: CacheError, Err: statErr}
	}

	data, readErr := os.ReadFile(path)
	if readErr != nil {
		if errors.Is(readErr, os.ErrNotExist) {
			return &CacheResult{State: CacheMissing}
		}
		return &CacheResult{State: CacheError, Err: readErr}
	}

	if time.Since(info.ModTime()) > cacheTTL {
		return &CacheResult{State: CacheStale, Data: data}
	}
	return &CacheResult{State: CacheFresh, Data: data}
}

func (c *Cache) Write(name string, data []byte) error {
	return c.WriteWith(name, func(f *os.File) error {
		_, err := f.Write(data)
		return err
	})
}

func (c *Cache) WriteWith(name string, write func(*os.File) error) error {
	dest := filepath.Join(c.dir, name)
	tmp := dest + ".tmp"

	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0660)
	if err != nil {
		return err
	}

	defer os.Remove(tmp)
	defer f.Close()

	if err = write(f); err != nil {
		return err
	}
	if err = f.Sync(); err != nil {
		return err
	}

	return os.Rename(tmp, dest)
}
