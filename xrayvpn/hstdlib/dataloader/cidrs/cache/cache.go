package cache

import (
	"bufio"
	"bytes"
	"fmt"

	datacache "github.com/realglebivanov/hstd/hstdlib/dataloader/cache"
)

type Cache struct {
	store *datacache.Cache
}

type ReadStatus int

const (
	Fresh ReadStatus = iota
	Stale
	Missing
	Error
)

type ReadResult struct {
	Status ReadStatus
	CIDRs  []string
	Err    error
}

func New(store *datacache.Cache) *Cache {
	return &Cache{store: store}
}

func (c *Cache) Read(srcName string) *ReadResult {
	name := cacheName(srcName)
	cr := c.store.Read(name)
	switch cr.State {
	case datacache.CacheStale:
		cidrs, err := unmarshalCIDRs(cr.Data)
		if err != nil {
			return &ReadResult{Status: Error, Err: err}
		}
		return &ReadResult{Status: Stale, CIDRs: cidrs}
	case datacache.CacheFresh:
		cidrs, err := unmarshalCIDRs(cr.Data)
		if err != nil {
			return &ReadResult{Status: Error, Err: err}
		}
		return &ReadResult{Status: Fresh, CIDRs: cidrs}
	case datacache.CacheMissing:
		return &ReadResult{Status: Missing}
	case datacache.CacheError:
		return &ReadResult{Status: Error, Err: fmt.Errorf("read %s cache: %w", srcName, cr.Err)}
	default:
		return &ReadResult{Status: Error, Err: fmt.Errorf("unexpected cache state %d for %s", cr.State, srcName)}
	}
}

func (c *Cache) Write(srcName string, cidrs []string) error {
	return c.store.Write(cacheName(srcName), marshalCIDRs(cidrs))
}

func cacheName(srcName string) string {
	return "ru_cidrs_" + srcName + ".txt"
}

func unmarshalCIDRs(data []byte) ([]string, error) {
	var result []string
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		if line := scanner.Text(); line != "" {
			result = append(result, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan CIDRs: %w", err)
	}
	return result, nil
}

func marshalCIDRs(cidrs []string) []byte {
	var buf bytes.Buffer
	for _, c := range cidrs {
		buf.WriteString(c)
		buf.WriteByte('\n')
	}
	return buf.Bytes()
}
