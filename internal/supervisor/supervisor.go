package supervisor

import (
	"fmt"
	"log"
	"sync"

	"github.com/realglebivanov/xray-vpn/internal/config"
	"github.com/realglebivanov/xray-vpn/internal/routing"
	core "github.com/xtls/xray-core/core"
	_ "github.com/xtls/xray-core/main/distro/all"
)

type supervisor struct {
	mu       sync.Mutex
	instance *core.Instance
	tun      *routing.Tunnel
	running  bool
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

	if err := config.RefreshGeodata(); err != nil {
		return fmt.Errorf("refresh geodata failed: %v", err)
	}
	if _, err := config.RefreshRuCIDRs(); err != nil {
		return fmt.Errorf("refresh CIDRs failed: %v", err)
	}
	if !s.running {
		log.Printf("data refreshed (not running, skipping restart)")
		return nil
	}

	log.Println("data refreshed, restarting with new data ...")
	return s.startLocked()
}

func (s *supervisor) startLocked() error {
	if s.running {
		if err := s.stopLocked(); err != nil {
			return err
		}
	}

	if err := s.startXRay(); err != nil {
		return err
	}
	if err := s.setUpTunnel(); err != nil {
		return err
	}

	s.running = true
	log.Printf("tunnel set up: %v - %v", s.tun.Gw.IP, s.tun.TunAddr.IP)
	return nil
}

func (s *supervisor) startXRay() error {
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
		return fmt.Errorf("start xray-core: %w", err)
	}
	s.instance = instance

	return nil
}

func (s *supervisor) setUpTunnel() error {
	tun, err := routing.SetUpTunnel()
	s.tun = tun

	if err != nil {
		if sErr := s.stopLocked(); sErr != nil {
			log.Printf("stopLocked err: %v", err)
		}
		return fmt.Errorf("tunnel set up: %w", err)
	}

	return nil
}

func (s *supervisor) stopLocked() error {
	if s.instance != nil {
		if err := s.instance.Close(); err != nil {
			return err
		}
		s.instance = nil
		log.Println("stopped xray-core")
	}

	if s.tun != nil {
		if err := routing.TearDownTunnel(s.tun); err != nil {
			return err
		}
		s.tun = nil
		log.Println("tunnel is down")
	}

	s.running = false

	return nil
}
