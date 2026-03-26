package cidrs

import (
	"fmt"
	"log"
)

func Load() ([]string, error) {
	var allCIDRs []string
	var missingSrcs []source

	for _, src := range sources {
		r := readOrRefresh(&src)
		switch r.status {
		case readOk:
			allCIDRs = append(allCIDRs, r.cidrs...)
		case readMissing:
			missingSrcs = append(missingSrcs, src)
		case readError:
			return nil, fmt.Errorf("read or cache %s: %w", src.Name, r.err)
		}
	}

	missingCIDRs, err := refreshSources(missingSrcs)

	if err != nil {
		log.Printf("refresh missing sources: %v", err)
		return nil, err
	}

	return dedup(append(allCIDRs, missingCIDRs...)), nil
}

func Refresh() error {
	if _, err := refreshSources(sources); err != nil {
		log.Printf("refresh cidrs: %v", err)
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
