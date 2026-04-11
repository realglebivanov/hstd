package cidrs

import (
	"errors"
	"log/slog"

	"github.com/realglebivanov/hstd/hstdlib/dataloader/cidrs/client"
)

type LoadResult struct {
	CIDRs  []string
	loader *Loader
	stale  []client.Source
}

func (d *LoadResult) ShouldRefresh() bool {
	return len(d.stale) > 0
}

func (d *LoadResult) Refresh() {
	if len(d.stale) == 0 {
		return
	}

	_, errs := client.FetchSources(d.stale, d.loader.fetchAndCache)
	if len(errs) == len(d.stale) {
		slog.Warn("fetch and cache at least one cidr source", "err", errors.Join(errs...))
		return
	}

	select {
	case d.loader.Update <- struct{}{}:
	default:
	}
}
