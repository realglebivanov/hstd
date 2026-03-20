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

func main() {
	log.SetFlags(log.Ltime)

	if err := hstdlib.CheckCap(unix.CAP_NET_ADMIN); err != nil {
		log.Fatalf("no CAP_NET_ADMIN capability: %v", err)
	}

	tun, err := startEngine()
	if err != nil {
		log.Fatalf("start: %v", err)
	}
	defer stopEngine(tun)
	log.Printf("tunnel up: %v → %v", tun.Gw.IP, tun.TunAddr.IP)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
	<-sigCh
	log.Println("shutting down ...")
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
		stopEngine(nil)
		return nil, fmt.Errorf("tunnel: %w", err)
	}

	return tun, nil
}

func stopEngine(tun *routing.Tunnel) {
	engine.Stop()
	if tun == nil {
		return
	}
	if err := routing.TearDownTunnel(tun); err != nil {
		log.Printf("tunnel teardown: %v", err)
	}
}
