package main

import (
	"fmt"
	"log"
	"os"

	"github.com/realglebivanov/hstd/hstdlib"
	"github.com/realglebivanov/hstd/xrayvpnd/internal/supervisor"
	"golang.org/x/sys/unix"
)

func main() {
	log.SetFlags(log.Ltime)

	if err := hstdlib.CheckCap(unix.CAP_NET_ADMIN); err != nil {
		log.Fatalf("no CAP_NET_ADMIN capability: %v", err)
	}

	if err := os.WriteFile(hstdlib.XrayVpnPIDFile, fmt.Appendf(nil, "%d", os.Getpid()), 0644); err != nil {
		log.Fatalf("write pid file: %v", err)
	}

	defer os.Remove(hstdlib.XrayVpnPIDFile)

	if err := supervisor.Run(); err != nil {
		log.Fatalf("fatal: %v", err)
	}
}
