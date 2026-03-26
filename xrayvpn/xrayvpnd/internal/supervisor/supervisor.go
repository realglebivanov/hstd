package supervisor

import (
	"fmt"
	"log"
	"sync"

	"github.com/realglebivanov/hstd/xrayvpnd/internal/cidrs"
	"github.com/realglebivanov/hstd/xrayvpnd/internal/config"
	"github.com/realglebivanov/hstd/xrayvpnd/internal/geodata"
	core "github.com/xtls/xray-core/core"
	_ "github.com/xtls/xray-core/main/distro/all"
)

type supervisor struct {
	mu       sync.Mutex
	instance *core.Instance
}

func (s *supervisor) start() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.startLocked()
}

func (s *supervisor) stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.stopLocked()
}

func (s *supervisor) refresh() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := geodata.Refresh(); err != nil {
		return fmt.Errorf("refresh geodata failed: %v", err)
	}

	if err := cidrs.Refresh(); err != nil {
		return fmt.Errorf("refresh cidrs failed: %v", err)
	}

	if s.instance == nil {
		log.Printf("data refreshed (not running, skipping restart)")
		return nil
	}

	log.Println("data refreshed, restarting with new data ...")
	return s.startLocked()
}

func (s *supervisor) startLocked() error {
	if err := s.stopLocked(); err != nil {
		return err
	}

	coreConfig, err := config.BuildCoreConfig()
	if err != nil {
		return fmt.Errorf("build xray-core config: %w", err)
	}

	log.Println("starting xray-core ...")
	instance, err := core.New(coreConfig)
	if err != nil {
		return fmt.Errorf("create xray-core: %w", err)
	}
	if err := instance.Start(); err != nil {
		instance.Close()
		return fmt.Errorf("start xray-core: %w", err)
	}
	s.instance = instance

	log.Println("xray-core started")
	return nil
}

func (s *supervisor) stopLocked() error {
	if s.instance == nil {
		return nil
	}

	if err := s.instance.Close(); err != nil {
		return err
	}

	s.instance = nil
	log.Println("stopped xray-core")

	return nil
}
