package supervisor

import (
	"fmt"
	"log"

	"github.com/realglebivanov/hstd/hstdlib"
	"github.com/realglebivanov/hstd/tun2socksd/internal/routing"
	"github.com/xjasonlyu/tun2socks/v2/engine"
)

type Supervisor struct {
	tun *routing.Tunnel
}

func (c *Supervisor) Start() error {
	if err := c.Stop(); err != nil {
		return err
	}

	tun, err := startEngine()
	if err != nil {
		return err
	}

	c.tun = tun
	log.Printf("tunnel up: %v → %v", tun.Gw.IP, tun.TunAddr.IP)
	return nil
}

func (c *Supervisor) Stop() error {
	if c.tun == nil {
		return nil
	}

	if err := stopEngine(c.tun); err != nil {
		return err
	}

	c.tun = nil
	log.Println("tunnel down")
	return nil
}

func startEngine() (*routing.Tunnel, error) {
	engine.Insert(&engine.Key{
		Device:   routing.TunDev,
		Proxy:    fmt.Sprintf("socks5://%s:%d", hstdlib.SocksHost, hstdlib.SocksPort),
		MTU:      routing.TunMTU,
		LogLevel: "warn",
	})

	engine.Start()

	tun, err := routing.SetUpTunnel()
	if err != nil {
		if stopErr := stopEngine(tun); stopErr != nil {
			return nil, fmt.Errorf("tunnel: %w (cleanup: %v)", err, stopErr)
		}
		return nil, fmt.Errorf("tunnel: %w", err)
	}

	return tun, nil
}

func stopEngine(tun *routing.Tunnel) error {
	engine.Stop()
	if tun == nil {
		return nil
	}
	if err := routing.TearDownTunnel(tun); err != nil {
		return fmt.Errorf("tunnel teardown: %w", err)
	}
	return nil
}
