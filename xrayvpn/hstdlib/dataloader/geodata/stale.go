package geodata

import "log/slog"

type Stale struct {
	stale  []geoFile
	loader *Loader
}

func (r *Stale) Refresh(notify chan<- struct{}) {
	if len(r.stale) == 0 {
		return
	}

	for _, f := range r.stale {
		if err := r.loader.fetchAndCache(f); err != nil {
			slog.Warn("background geodata refresh failed", "name", f.name, "err", err)
		}
	}

	select {
	case notify <- struct{}{}:
	default:
	}
}
