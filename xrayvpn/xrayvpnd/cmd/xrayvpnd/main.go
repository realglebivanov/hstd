package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/realglebivanov/hstd/hstdlib"
	"github.com/realglebivanov/hstd/xrayvpnd/internal/supervisor"
	"golang.org/x/sys/unix"
)

func main() {
	if err := hstdlib.CheckCap(unix.CAP_NET_ADMIN); err != nil {
		slog.Error("no CAP_NET_ADMIN capability", "err", err)
		os.Exit(1)
	}

	if err := os.WriteFile(hstdlib.XrayVpnPIDFile, fmt.Appendf(nil, "%d", os.Getpid()), 0644); err != nil {
		slog.Error("write pid file", "err", err)
		os.Exit(1)
	}

	defer os.Remove(hstdlib.XrayVpnPIDFile)

	if err := supervisor.Run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}
