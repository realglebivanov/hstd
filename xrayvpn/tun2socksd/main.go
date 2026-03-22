package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/realglebivanov/hstd/hstdlib"
	s "github.com/realglebivanov/hstd/tun2socksd/internal/supervisor"
	"golang.org/x/sys/unix"
)

func main() {
	log.SetFlags(log.Ltime)

	if err := hstdlib.CheckCap(unix.CAP_NET_ADMIN); err != nil {
		log.Fatalf("no CAP_NET_ADMIN capability: %v", err)
	}

	if err := os.WriteFile(hstdlib.Tun2SocksPIDFile, fmt.Appendf(nil, "%d", os.Getpid()), 0644); err != nil {
		log.Fatalf("write pid file: %v", err)
	}
	defer os.Remove(hstdlib.Tun2SocksPIDFile)

	c := &s.Supervisor{}
	if err := c.Start(); err != nil {
		log.Fatalf("start: %v", err)
	}

	handleSignals(c)
}

func handleSignals(c *s.Supervisor) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGUSR1, syscall.SIGUSR2, syscall.SIGTERM, syscall.SIGINT)

	for sig := range sigCh {
		switch sig {
		case syscall.SIGUSR2:
			log.Println("SIGUSR2: (re)starting tunnel ...")
			if err := c.Start(); err != nil {
				log.Printf("(re)start failed: %v", err)
			}
		case syscall.SIGUSR1:
			log.Println("SIGUSR1: stopping tunnel ...")
			if err := c.Stop(); err != nil {
				log.Printf("stop failed: %v", err)
			}
		case syscall.SIGTERM, syscall.SIGINT:
			log.Println("shutting down ...")
			if err := c.Stop(); err != nil {
				log.Printf("stop failed: %v", err)
			}
			return
		}
	}
}
