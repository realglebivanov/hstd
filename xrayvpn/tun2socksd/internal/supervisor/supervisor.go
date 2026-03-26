package supervisor

import (
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/realglebivanov/hstd/hstdlib"
	"github.com/realglebivanov/hstd/tun2socksd/internal/tunnel"
	"github.com/xjasonlyu/tun2socks/v2/engine"
)

var mu sync.Mutex

type Supervisor struct {
	tun *tunnel.Tunnel
}

func (s *Supervisor) Start() error {
	mu.Lock()
	defer mu.Unlock()

	if err := s.stopLocked(); err != nil {
		return err
	}

	tun, err := startEngine()
	if err != nil {
		return err
	}

	s.tun = tun
	log.Printf("tunnel up: %v → %v", tun.DefaultGwAddr(), tun.TunAddr())
	return nil
}

func (s *Supervisor) Stop() error {
	mu.Lock()
	defer mu.Unlock()
	return s.stopLocked()
}

func (s *Supervisor) stopLocked() error {
	if s.tun == nil {
		return nil
	}

	if err := stopEngine(s.tun); err != nil {
		return err
	}

	s.tun = nil
	log.Println("tunnel down")
	return nil
}

func startEngine() (*tunnel.Tunnel, error) {
	engine.Insert(&engine.Key{
		Device:   hstdlib.TunDev,
		Proxy:    fmt.Sprintf("socks5://%s:%d", hstdlib.SocksHost, hstdlib.SocksPort),
		MTU:      hstdlib.TunMTU,
		LogLevel: "warn",
	})

	engine.Start()

	tun, err := tunnel.New()
	if err != nil {
		engine.Stop()
		return nil, fmt.Errorf("tunnel new: %w", err)
	}

	if err := tun.SetUp(); err != nil {
		return nil, errors.Join(err, stopEngine(tun))
	}

	return tun, nil
}

func stopEngine(tun *tunnel.Tunnel) error {
	engine.Stop()
	if err := tun.TearDown(); err != nil {
		return fmt.Errorf("tunnel teardown: %w", err)
	}
	return nil
}
