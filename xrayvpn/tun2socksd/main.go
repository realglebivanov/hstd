package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/realglebivanov/hstd/hstdlib"
	"github.com/realglebivanov/hstd/tun2socksd/routing"
	"github.com/xjasonlyu/tun2socks/v2/engine"
	"golang.org/x/sys/unix"
)

type controller struct {
	tun *routing.Tunnel
}

func main() {
	log.SetFlags(log.Ltime)

	if err := hstdlib.CheckCap(unix.CAP_NET_ADMIN); err != nil {
		log.Fatalf("no CAP_NET_ADMIN capability: %v", err)
	}

	if err := os.WriteFile(hstdlib.Tun2SocksPIDFile, fmt.Appendf(nil, "%d", os.Getpid()), 0644); err != nil {
		log.Fatalf("write pid file: %v", err)
	}
	defer os.Remove(hstdlib.Tun2SocksPIDFile)

	c := &controller{}
	if err := c.start(); err != nil {
		log.Fatalf("start: %v", err)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGUSR1, syscall.SIGUSR2, syscall.SIGTERM, syscall.SIGINT)
	handleSignals(c, sigCh)
}

func handleSignals(c *controller, sigCh chan os.Signal) {
	for sig := range sigCh {
		switch sig {
		case syscall.SIGUSR2:
			log.Println("SIGUSR2: (re)starting tunnel ...")
			if err := c.start(); err != nil {
				log.Printf("(re)start failed: %v", err)
			}
		case syscall.SIGUSR1:
			log.Println("SIGUSR1: stopping tunnel ...")
			if err := c.stop(); err != nil {
				log.Printf("stop failed: %v", err)
			}
		case syscall.SIGTERM, syscall.SIGINT:
			log.Println("shutting down ...")
			if err := c.stop(); err != nil {
				log.Printf("stop failed: %v", err)
			}
			return
		}
	}
}

func (c *controller) start() error {
	if c.tun != nil {
		if err := c.stop(); err != nil {
			return err
		}
	}

	tun, err := startEngine()
	if err != nil {
		return err
	}

	c.tun = tun
	log.Printf("tunnel up: %v → %v", tun.Gw.IP, tun.TunAddr.IP)
	return nil
}

func (c *controller) stop() error {
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
