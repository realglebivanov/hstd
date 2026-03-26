package cidrs

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"strings"

	"github.com/realglebivanov/hstd/xrayvpnd/internal/cache"
)

type readStatus int

const (
	readOk readStatus = iota
	readMissing
	readError
)

type readResult struct {
	status readStatus
	cidrs  []string
	err    error
}

func readOrRefresh(src *source) *readResult {
	cr := cache.Read(cacheName(src))
	switch cr.State {
	case cache.CacheStale:
		log.Printf("will refresh %s CIDRs in background", src.Name)
		go fetchAndCacheSource(src)
		fallthrough
	case cache.CacheFresh:
		cidrs, err := unmarshalCIDRs(cr.Data)
		if err != nil {
			return &readResult{status: readError, err: err}
		}
		return &readResult{status: readOk, cidrs: cidrs}
	case cache.CacheMissing:
		return &readResult{status: readMissing}
	case cache.CacheError:
		return &readResult{status: readError, err: fmt.Errorf("read %s cache: %w", src.Name, cr.Err)}
	default:
		return &readResult{status: readError, err: fmt.Errorf("unexpected cache state %d for %s", cr.State, src.Name)}
	}
}

type sourceResult struct {
	src   *source
	cidrs []string
	err   error
}

func refreshSources(srcs []source) ([]string, error) {
	ch := make(chan *sourceResult, len(srcs))
	var cidrs []string

	for _, src := range srcs {
		go func(src *source) {
			ch <- fetchAndCacheSource(src)
		}(&src)
	}

	var errs []error
	for range srcs {
		r := <-ch
		if r.err == nil {
			cidrs = append(cidrs, r.cidrs...)
			continue
		}
		log.Printf("warning: failed to fetch %s: %v", r.src.Name, r.err)
		errs = append(errs, r.err)
	}

	if len(errs) != 0 {
		return nil, fmt.Errorf("%d/%d sources failed", len(errs), len(srcs))
	}

	return cidrs, nil
}

func fetchAndCacheSource(src *source) *sourceResult {
	cidrs, err := tryToFetchSource(src)
	if err != nil {
		return &sourceResult{src: src, cidrs: cidrs, err: err}
	}
	if err := cache.Write(cacheName(src), []byte(strings.Join(cidrs, "\n")+"\n")); err != nil {
		log.Printf("warning: failed to write %s cache: %v", src.Name, err)
		return &sourceResult{src: src, err: err}
	}
	log.Printf("wrote %d CIDRs from %s to cache", len(cidrs), src.Name)
	return &sourceResult{src: src, cidrs: cidrs}
}

func unmarshalCIDRs(data []byte) ([]string, error) {
	var cidrs []string
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		if line := scanner.Text(); line != "" {
			cidrs = append(cidrs, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan CIDRs: %w", err)
	}
	return cidrs, nil
}

func cacheName(src *source) string {
	return "ru_cidrs_" + src.Name + ".txt"
}
