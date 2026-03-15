// xray-vpnd is the long-running daemon embedding xray-core.
//
// Signals:
//   - SIGUSR2: (re)start xray + routes (daemon stays alive)
//   - SIGUSR1: stop xray + routes (daemon stays alive)
//   - SIGTERM/SIGINT: full shutdown (daemon exits)

package main

import (
	"fmt"
	"log"
	"os"

	"github.com/realglebivanov/xray-vpn/internal/supervisor"
	"golang.org/x/sys/unix"
)

const pidFile = "/run/xray-vpn/xray-vpn.pid"

func main() {
	log.SetFlags(log.Ltime)

	hdr := unix.CapUserHeader{Version: unix.LINUX_CAPABILITY_VERSION_3}
	data := unix.CapUserData{}

	if err := unix.Capget(&hdr, &data); err != nil {
		log.Fatalf("unix.Capget failed: %v", err)
		return
	}

	if data.Effective&(1<<unix.CAP_NET_ADMIN) == 0 && os.Getuid() != 0 {
		log.Fatal("no CAP_NET_ADMIN capability")
		return
	}

	if err := os.WriteFile(pidFile, fmt.Appendf(nil, "%d", os.Getpid()), 0644); err != nil {
		log.Fatalf("write pid file: %v", err)
		return
	}
	defer os.Remove(pidFile)

	if err := supervisor.Run(); err != nil {
		log.Fatalf("fatal: %v", err)
	}
}
