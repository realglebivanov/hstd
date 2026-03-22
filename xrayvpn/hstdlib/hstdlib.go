package hstdlib

import (
	"fmt"
	"os"
	"strconv"

	"golang.org/x/sys/unix"
)

var (
	SocksHost        = EnvOr("SOCKS_HOST", "127.0.0.1")
	SocksPort        = EnvOrUint32("SOCKS_PORT", 1080)
	XrayVpnPIDFile   = "/run/xrayvpn/xrayvpnd.pid"
	Tun2SocksPIDFile = "/run/xrayvpn/tun2socksd.pid"
	XrayOutMark      = EnvOrUint32("XRAY_OUT_MARK", 0x1f)
	XrayTrafficMark  = EnvOrUint32("XRAY_TRAFFIC_MARK", 0x1337)
)

func CheckCap(cap int) error {
	hdr := unix.CapUserHeader{Version: unix.LINUX_CAPABILITY_VERSION_3}
	data := unix.CapUserData{}

	if err := unix.Capget(&hdr, &data); err != nil {
		return fmt.Errorf("unix.Capget: %v", err)
	}
	if data.Effective&(1<<cap) == 0 && os.Getuid() != 0 {
		return fmt.Errorf("neither required capability nor root")
	}

	return nil
}

func EnvOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func EnvOrInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func EnvOrUint32(key string, fallback uint32) uint32 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.ParseUint(v, 10, 32)
	if err != nil {
		return fallback
	}
	return uint32(n)
}
