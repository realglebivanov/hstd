package geodata

import (
	"errors"
	"log/slog"
)

type Stale struct {
	stale  []geoFile
	loader *Loader
}

func (r *Stale) ShouldRefresh() bool {
	return len(r.stale) > 0
}

func (r *Stale) Refresh() {
	if len(r.stale) == 0 {
		return
	}

	errs := r.loader.fetchAndCacheMany(r.stale)
	if len(r.stale) == len(errs) {
		slog.Warn("fetch and cache at least one stale geofile", "err", errors.Join(errs...))
		return
	}

	select {
	case r.loader.Notify <- struct{}{}:
	default:
	}
}
