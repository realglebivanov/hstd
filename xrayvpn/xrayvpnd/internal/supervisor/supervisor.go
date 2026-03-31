package supervisor

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/realglebivanov/hstd/xrayvpnd/internal/config"
	"github.com/realglebivanov/hstd/xrayvpnd/internal/dataloader"
	core "github.com/xtls/xray-core/core"
	_ "github.com/xtls/xray-core/main/distro/all"
)

type Supervisor struct {
	mu        sync.Mutex
	wg        sync.WaitGroup
	instance  *core.Instance
	RefreshCh chan struct{}
}

func New() *Supervisor {
	return &Supervisor{RefreshCh: make(chan struct{}, 1)}
}

func (s *Supervisor) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.startLocked()
}

func (s *Supervisor) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stopLocked()
}

func (s *Supervisor) Refresh() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := dataloader.RefreshAll(); err != nil {
		return fmt.Errorf("refresh failed: %v", err)
	}

	if s.instance == nil {
		slog.Info("data refreshed (not running, skipping restart)")
		return nil
	}

	slog.Info("data refreshed, restarting with new data ...")
	return s.startLocked()
}

func (s *Supervisor) startLocked() error {
	if err := s.stopLocked(); err != nil {
		return err
	}

	data, err := dataloader.Load()
	if err != nil {
		return fmt.Errorf("load data: %w", err)
	}

	coreConfig, err := config.BuildCoreConfig(data.RuCIDRs)
	if err != nil {
		return fmt.Errorf("build xray-core config: %w", err)
	}

	slog.Info("starting xray-core ...")
	instance, err := core.New(coreConfig)
	if err != nil {
		return fmt.Errorf("create xray-core: %w", err)
	}
	if err := instance.Start(); err != nil {
		instance.Close()
		return fmt.Errorf("start xray-core: %w", err)
	}
	s.instance = instance

	slog.Info("xray-core started")

	if data.Stale.Any() {
		s.wg.Go(func() {
			data.Stale.Refresh()
			select {
			case s.RefreshCh <- struct{}{}:
			default:
			}
		})
	}

	return nil
}

func (s *Supervisor) stopLocked() error {
	s.wg.Wait()

	if s.instance == nil {
		return nil
	}

	if err := s.instance.Close(); err != nil {
		return err
	}

	s.instance = nil
	slog.Info("stopped xray-core")

	return nil
}
