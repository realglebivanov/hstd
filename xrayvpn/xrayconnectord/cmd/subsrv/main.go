package main

import (
	"log/slog"
	"os"

	"github.com/realglebivanov/hstd/hstdlib"
	"github.com/realglebivanov/hstd/xrayconnectord/internal/server"
)

func main() {
	rootSecret := hstdlib.MustEnvHex("SECRET")

	s, err := server.New(rootSecret)
	if err != nil {
		slog.Error("init server", "err", err)
		os.Exit(1)
	}
	defer s.Stop()

	s.Start()
}
