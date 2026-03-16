package supervisor

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/realglebivanov/xray-vpn/internal/config"
	"github.com/realglebivanov/xray-vpn/internal/routing"
	core "github.com/xtls/xray-core/core"
	_ "github.com/xtls/xray-core/main/distro/all"
)

type Supervisor struct {
	mu       sync.Mutex
	instance *core.Instance
	tun      *routing.Tunnel
	running  bool
}

func (s *Supervisor) start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		s.stopLocked()
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
		return fmt.Errorf("start xray-core: %w", err)
	}
	s.instance = instance

	tun, err := routing.SetUpTunnel()
	if err != nil {
		s.stopLocked()
		return fmt.Errorf("tunnel set up: %w", err)
	}
	s.tun = tun

	log.Printf("tunnel set up: %s - %s", tun.Gw.IP.String(), tun.TunAddr.IP.String())

	s.running = true
	return nil
}

func (s *Supervisor) stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stopLocked()
}

func (s *Supervisor) refresh() {
	if err := config.RefreshGeodata(); err != nil {
		log.Printf("refresh geodata failed: %v", err)
		return
	}
	if _, err := config.RefreshRuCIDRs(); err != nil {
		log.Printf("refresh CIDRs failed: %v", err)
		return
	}
	if !s.running {
		log.Println("data refreshed (not running, skipping restart)")
		return
	}
	log.Println("data refreshed, restarting with new data ...")
	if err := s.start(); err != nil {
		log.Printf("restart after refresh failed: %v", err)
	}
}

func (s *Supervisor) stopLocked() {
	if s.tun != nil {
		routing.TearDownTunnel(s.tun)
	}

	if s.instance != nil {
		s.instance.Close()
		s.instance = nil
	}

	for range 20 {
		if !routing.LinkExists(routing.TunDev) {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	s.running = false
	log.Println("stopped xray-core")
}
