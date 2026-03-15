package supervisor

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func Run() error {
	s := &Supervisor{}

	if err := s.start(); err != nil {
		return fmt.Errorf("initial start: %w", err)
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGUSR1, syscall.SIGUSR2, syscall.SIGTERM, syscall.SIGINT)

	for sig := range sigCh {
		switch sig {
		case syscall.SIGUSR2:
			log.Println("SIGUSR2: (re)starting ...")
			s.stop()
			if err := s.start(); err != nil {
				log.Printf("start failed: %v", err)
			}

		case syscall.SIGUSR1:
			log.Println("SIGUSR1: stopping ...")
			s.stop()

		case syscall.SIGTERM, syscall.SIGINT:
			log.Println("shutting down ...")
			s.stop()
			return nil
		}
	}
	return nil
}
