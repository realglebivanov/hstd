package main

import (
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/realglebivanov/hstd/hstdlib"
	s "github.com/realglebivanov/hstd/tun2socksd/internal/supervisor"
	"golang.org/x/sys/unix"
)

func main() {
	if err := hstdlib.CheckCap(unix.CAP_NET_ADMIN); err != nil {
		slog.Error("no CAP_NET_ADMIN capability", "err", err)
		os.Exit(1)
	}

	if err := os.WriteFile(hstdlib.Tun2SocksPIDFile, fmt.Appendf(nil, "%d", os.Getpid()), 0644); err != nil {
		slog.Error("write pid file", "err", err)
		os.Exit(1)
	}
	defer os.Remove(hstdlib.Tun2SocksPIDFile)

	c := &s.Supervisor{}
	if err := c.Start(); err != nil {
		slog.Error("start", "err", err)
		os.Exit(1)
	}

	handleSignals(c)
}

func handleSignals(c *s.Supervisor) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGUSR1, syscall.SIGUSR2, syscall.SIGTERM, syscall.SIGINT)

	for sig := range sigCh {
		switch sig {
		case syscall.SIGUSR2:
			slog.Info("SIGUSR2: (re)starting tunnel ...")
			if err := c.Start(); err != nil {
				slog.Error("(re)start failed", "err", err)
			}
		case syscall.SIGUSR1:
			slog.Info("SIGUSR1: stopping tunnel ...")
			if err := c.Stop(); err != nil {
				slog.Error("stop failed", "err", err)
			}
		case syscall.SIGTERM, syscall.SIGINT:
			slog.Info("shutting down ...")
			if err := c.Stop(); err != nil {
				slog.Error("stop failed", "err", err)
			}
			return
		}
	}
}
