package supervisor

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"
)

func Run() error {
	s := &supervisor{}

	if err := s.start(); err != nil {
		return fmt.Errorf("initial start: %w", err)
	}

	if err := sdNotify("READY=1"); err != nil {
		return errors.Join(err, s.stop())
	}

	return handleSignals(s)
}

func handleSignals(s *supervisor) error {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGUSR1, syscall.SIGUSR2, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGINT)

	for sig := range sigCh {
		switch sig {
		case syscall.SIGUSR2:
			slog.Info("SIGUSR2: (re)starting ...")
			if err := s.start(); err != nil {
				slog.Error("(re)start failed", "err", err)
			}

		case syscall.SIGUSR1:
			slog.Info("SIGUSR1: stopping ...")
			if err := s.stop(); err != nil {
				slog.Error("stop failed", "err", err)
			}

		case syscall.SIGHUP:
			slog.Info("SIGHUP: refreshing RU CIDRs and geodata ...")
			if err := s.refresh(); err != nil {
				slog.Error("refresh failed", "err", err)
			}

		case syscall.SIGTERM, syscall.SIGINT:
			slog.Info("shutting down ...")
			return s.stop()
		}
	}
	return nil
}

func sdNotify(state string) error {
	addr := os.Getenv("NOTIFY_SOCKET")
	if addr == "" {
		return fmt.Errorf("no NOTIFY_SOCKET")
	}
	conn, err := net.Dial("unixgram", addr)
	if err != nil {
		return fmt.Errorf("sd_notify: %v", err)
	}
	defer conn.Close()
	if _, err := conn.Write([]byte(state)); err != nil {
		return fmt.Errorf("sd_notify: %v", err)
	}
	return nil
}
