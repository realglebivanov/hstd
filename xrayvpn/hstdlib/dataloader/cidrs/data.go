package cidrs

import (
	"errors"
	"log/slog"

	"github.com/realglebivanov/hstd/hstdlib/dataloader/cidrs/client"
)

type Data struct {
	CIDRs  []string
	loader *Loader
	stale  []*client.Source
}

func (d *Data) ShouldRefresh() bool {
	return len(d.stale) > 0
}

func (d *Data) Refresh() {
	if len(d.stale) == 0 {
		return
	}

	_, errs := d.loader.refreshSources(d.stale)
	if len(errs) == len(d.stale) {
		slog.Warn("fetch and cache at least one cidr source", "err", errors.Join(errs...))
		return
	}

	select {
	case d.loader.Update <- struct{}{}:
	default:
	}
}
