package cidrs

import (
	"fmt"
	"log/slog"
)

func Load() (cidrs []string, stale []Source, err error) {
	var missingSrcs []Source

	for _, src := range sources {
		r := readOrRefresh(&src)
		switch r.status {
		case readOk:
			cidrs = append(cidrs, r.cidrs...)
		case readStale:
			cidrs = append(cidrs, r.cidrs...)
			stale = append(stale, src)
		case readMissing:
			missingSrcs = append(missingSrcs, src)
		case readError:
			return nil, nil, fmt.Errorf("read or cache %s: %w", src.Name, r.err)
		}
	}

	missingCIDRs, err := refreshSources(missingSrcs)
	if err != nil {
		slog.Error("refresh missing sources", "err", err)
		return nil, nil, err
	}

	return dedup(append(cidrs, missingCIDRs...)), stale, nil
}

func RefreshSource(src Source) error {
	r := fetchAndCacheSource(&src)
	return r.err
}

func Refresh() error {
	if _, err := refreshSources(sources); err != nil {
		slog.Error("refresh cidrs", "err", err)
		return err
	}
	return nil
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
