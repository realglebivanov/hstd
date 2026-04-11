package cidrs

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"log/slog"

	"github.com/realglebivanov/hstd/hstdlib/dataloader/cache"
	"github.com/realglebivanov/hstd/hstdlib/dataloader/cidrs/client"
)

type Loader struct {
	cache  *cache.Cache
	Update chan struct{}
}

func NewLoader(cacheDir string) *Loader {
	return &Loader{
		cache:  cache.New(cacheDir),
		Update: make(chan struct{}, 1),
	}
}

func (l *Loader) Load() (*LoadResult, error) {
	var missingSrcs []client.Source
	var staleSrcs []client.Source
	var cidrs []string

	for _, src := range client.Sources {
		r := l.cache.Read(cacheName(src.Name))
		switch r.State {
		case cache.CacheFresh:
			slog.Info("using fresh CIDRs", "src", src.Name)
			parsed, err := parseCIDRs(r.Data)
			if err != nil {
				return nil, fmt.Errorf("parse cached %s: %w", src.Name, err)
			}
			cidrs = append(cidrs, parsed...)
		case cache.CacheStale:
			slog.Info("using stale CIDRs", "src", src.Name)
			parsed, err := parseCIDRs(r.Data)
			if err != nil {
				return nil, fmt.Errorf("parse cached %s: %w", src.Name, err)
			}
			staleSrcs = append(staleSrcs, src)
			cidrs = append(cidrs, parsed...)
		case cache.CacheMissing:
			missingSrcs = append(missingSrcs, src)
		case cache.CacheError:
			return nil, fmt.Errorf("read or cache %s: %w", src.Name, r.Err)
		}
	}

	missingSrcCIDRs, errs := client.FetchSources(client.Sources, l.fetchAndCache)
	if err := errors.Join(errs...); err != nil {
		slog.Error("refresh missing sources", "err", err)
		return nil, err
	}

	return &LoadResult{
		CIDRs:  dedup(append(cidrs, missingSrcCIDRs...)),
		stale:  staleSrcs,
		loader: l,
	}, nil
}

func FetchAll() ([]string, []error) {
	return client.FetchSources(client.Sources, client.FetchSource)
}

func (l *Loader) fetchAndCache(src *client.Source) *client.FetchResult {
	r := client.FetchSource(src)
	if r.Err != nil {
		return r
	}
	if err := l.cache.Write(cacheName(src.Name), marshalCIDRs(r.CIDRs)); err != nil {
		slog.Warn("failed to write cache", "src", src.Name, "err", err)
		return r
	}
	slog.Info("wrote CIDRs to cache", "count", len(r.CIDRs), "src", src.Name)
	return r
}

func cacheName(srcName string) string {
	return "ru_cidrs_" + srcName + ".txt"
}

func parseCIDRs(data []byte) ([]string, error) {
	var result []string
	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		if line := scanner.Text(); line != "" {
			result = append(result, line)
		}
	}
	return result, scanner.Err()
}

func marshalCIDRs(cidrs []string) []byte {
	var buf bytes.Buffer
	for _, c := range cidrs {
		buf.WriteString(c)
		buf.WriteByte('\n')
	}
	return buf.Bytes()
}

func dedup(cidrs []string) []string {
	seen := make(map[string]struct{}, len(cidrs))
	out := make([]string, 0, len(cidrs))
	for _, c := range cidrs {
		if _, ok := seen[c]; !ok {
			seen[c] = struct{}{}
			out = append(out, c)
		}
	}
	return out
}
