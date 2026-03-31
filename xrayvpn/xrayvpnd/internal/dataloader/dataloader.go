package dataloader

import (
	"errors"
	"fmt"
	"log/slog"
	"sync"

	"github.com/realglebivanov/hstd/xrayvpnd/internal/dataloader/cidrs"
	"github.com/realglebivanov/hstd/xrayvpnd/internal/dataloader/geodata"
)

type StaleData struct {
	gfs []geodata.GeoFile
	css []cidrs.Source
	wg  sync.WaitGroup
}

func (s *StaleData) Any() bool {
	return len(s.gfs) > 0 || len(s.css) > 0
}

func (s *StaleData) Refresh() {
	for _, f := range s.gfs {
		s.wg.Go(func() {
			if err := geodata.Download(f); err != nil {
				slog.Warn("background geodata refresh failed", "name", f.Name, "err", err)
			}
		})
	}
	for _, src := range s.css {
		s.wg.Go(func() {
			if err := cidrs.RefreshSource(src); err != nil {
				slog.Warn("background cidrs refresh failed", "src", src.Name, "err", err)
			}
		})
	}
	s.wg.Wait()
}

type Result struct {
	RuCIDRs []string
	Stale   *StaleData
}

func Load() (*Result, error) {
	staleGeo, err := geodata.Load()
	if err != nil {
		return nil, fmt.Errorf("load geodata: %w", err)
	}

	ruCIDRs, staleCIDRSrcs, err := cidrs.Load()
	if err != nil {
		return nil, fmt.Errorf("load ru CIDRs: %w", err)
	}

	return &Result{
		RuCIDRs: ruCIDRs,
		Stale:   &StaleData{gfs: staleGeo, css: staleCIDRSrcs},
	}, nil
}

func RefreshAll() error {
	return errors.Join(geodata.Refresh(), cidrs.Refresh())
}
