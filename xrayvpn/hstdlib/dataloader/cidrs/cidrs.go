package cidrs

import (
	"fmt"
	"log/slog"
	"sync"

	datacache "github.com/realglebivanov/hstd/hstdlib/dataloader/cache"
	"github.com/realglebivanov/hstd/hstdlib/dataloader/cidrs/cache"
	"github.com/realglebivanov/hstd/hstdlib/dataloader/cidrs/client"
)

type Loader struct {
	cache *cache.Cache
}

func NewLoader(c *datacache.Cache) *Loader {
	return &Loader{cache: cache.New(c)}
}

func (l *Loader) Load() (cidrs []string, err error) {
	var missingSrcs []client.Source

	for _, src := range client.Sources {
		r := l.cache.Read(src.Name)
		switch r.Status {
		case cache.Fresh:
			slog.Info("using fresh CIDRs", "src", src.Name)
			cidrs = append(cidrs, r.CIDRs...)
		case cache.Stale, cache.Missing:
			missingSrcs = append(missingSrcs, src)
		case cache.Error:
			return nil, fmt.Errorf("read or cache %s: %w", src.Name, r.Err)
		}
	}

	missingCIDRs, err := l.refreshSources(missingSrcs)
	if err != nil {
		slog.Error("refresh missing sources", "err", err)
		return nil, err
	}

	return dedup(append(cidrs, missingCIDRs...)), nil
}

type sourceResult struct {
	src   *client.Source
	cidrs []string
	err   error
}

func (l *Loader) refreshSources(srcs []client.Source) ([]string, error) {
	results := make([]*sourceResult, len(srcs))

	var wg sync.WaitGroup
	for i, src := range srcs {
		wg.Go(func() { results[i] = l.fetchAndCache(&src) })
	}
	wg.Wait()

	var allCIDRs []string
	var errs []error
	for _, r := range results {
		if r.err == nil {
			allCIDRs = append(allCIDRs, r.cidrs...)
			continue
		}
		slog.Warn("failed to fetch", "src", r.src.Name, "err", r.err)
		errs = append(errs, r.err)
	}

	if len(errs) > 0 {
		return nil, fmt.Errorf("%d/%d sources failed", len(errs), len(srcs))
	}

	return allCIDRs, nil
}

func (l *Loader) fetchAndCache(src *client.Source) *sourceResult {
	cidrs, err := client.FetchSource(src)
	if err != nil {
		return &sourceResult{src: src, err: err}
	}
	if err := l.cache.Write(src.Name, cidrs); err != nil {
		slog.Warn("failed to write cache", "src", src.Name, "err", err)
		return &sourceResult{src: src, err: err}
	}
	slog.Info("wrote CIDRs to cache", "count", len(cidrs), "src", src.Name)
	return &sourceResult{src: src, cidrs: cidrs}
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
