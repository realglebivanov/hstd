package supervisor

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/realglebivanov/hstd/hstdlib/dataloader/geodata"
	"github.com/realglebivanov/hstd/xrayvpnd/internal/config"
	"github.com/realglebivanov/hstd/xrayvpnd/internal/config/repo"
	"github.com/xtls/xray-core/common/platform"
	core "github.com/xtls/xray-core/core"
	_ "github.com/xtls/xray-core/main/distro/all"
)

type Supervisor struct {
	mu       sync.Mutex
	wg       sync.WaitGroup
	instance *core.Instance
	loader   *geodata.Loader
	db       *repo.DB
}

func New(db *repo.DB) *Supervisor {
	cacheDir := platform.GetAssetLocation("")
	loader := geodata.NewLoader(cacheDir)
	return &Supervisor{loader: loader, db: db}
}

func (s *Supervisor) Updates() chan struct{} {
	return s.loader.Notify
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

	if err := s.loader.Refresh(); err != nil {
		return fmt.Errorf("refresh failed: %v", err)
	}

	if s.instance == nil {
		slog.Info("geodata refreshed (not running, skipping restart)")
		return nil
	}

	slog.Info("geodata refreshed, restarting with new data ...")
	return s.startLocked()
}

func (s *Supervisor) startLocked() error {
	if err := s.stopLocked(); err != nil {
		return err
	}

	geoStale, err := s.loader.Load()
	if err != nil {
		return fmt.Errorf("load geodata: %w", err)
	}

	coreConfig, err := config.BuildCoreConfig(s.db)
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

	if geoStale.ShouldRefresh() {
		s.wg.Go(func() { geoStale.Refresh() })
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
