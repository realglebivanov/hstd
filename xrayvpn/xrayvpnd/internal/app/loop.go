package app

import (
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/realglebivanov/hstd/xrayvpnd/internal/config/repo"
	"github.com/realglebivanov/hstd/xrayvpnd/internal/supervisor"
)

func Run() error {
	db, err := repo.Open()
	if err != nil {
		return fmt.Errorf("open store: %w", err)
	}
	defer db.Close()

	s := supervisor.New(db)

	if err := s.Start(); err != nil {
		return fmt.Errorf("initial start: %w", err)
	}

	if err := sdNotify("READY=1"); err != nil {
		return errors.Join(err, s.Stop())
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGUSR1, syscall.SIGUSR2, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGINT)

	for {
		select {
		case <-s.Updates():
			slog.Info("background refresh done, restarting ...")
			if err := s.Start(); err != nil {
				slog.Error("restart after refresh failed", "err", err)
			}

		case sig := <-sigCh:
			if err, exit := handleSignal(s, sig); exit {
				return err
			}
		}
	}
}

func handleSignal(s *supervisor.Supervisor, sig os.Signal) (error, bool) {
	switch sig {
	case syscall.SIGUSR2:
		slog.Info("SIGUSR2: (re)starting ...")
		if err := s.Start(); err != nil {
			slog.Error("(re)start failed", "err", err)
		}

	case syscall.SIGUSR1:
		slog.Info("SIGUSR1: stopping ...")
		if err := s.Stop(); err != nil {
			slog.Error("stop failed", "err", err)
		}

	case syscall.SIGHUP:
		slog.Info("SIGHUP: refreshing RU CIDRs and geodata ...")
		if err := s.Refresh(); err != nil {
			slog.Error("refresh failed", "err", err)
		}

	case syscall.SIGTERM, syscall.SIGINT:
		slog.Info("shutting down ...")
		return s.Stop(), true
	}
	return nil, false
}

func sdNotify(state string) error {
	addr := os.Getenv("NOTIFY_SOCKET")
	if addr == "" {
		return fmt.Errorf("no NOTIFY_SOCKET")
	}
	conn, err := net.Dial("unixgram", addr)
	if err != nil {
		return fmt.Errorf("sd_notify: %w", err)
	}
	defer conn.Close()
	if _, err := conn.Write([]byte(state)); err != nil {
		return fmt.Errorf("sd_notify: %w", err)
	}
	return nil
}
