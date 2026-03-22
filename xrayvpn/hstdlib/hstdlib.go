package hstdlib

import (
	"fmt"
	"net"
	"os"
	"os/user"
	"strconv"

	"golang.org/x/sys/unix"
)

var (
	SocksHost        = EnvOr("SOCKS_HOST", "127.0.0.1")
	SocksPort        = EnvOrUint32("SOCKS_PORT", 1080)
	ApdCIDR          = EnvOr("APD_CIDR", "")
	TransmissionUser = EnvOr("TRANSMISSION_USER", "debian-transmission")
	XrayVpnPIDFile   = "/run/xrayvpn/xrayvpnd.pid"
	Tun2SocksPIDFile = "/run/xrayvpn/tun2socksd.pid"
	XrayOutMark      = EnvOrUint32("XRAY_OUT_MARK", 0x1f)
	XrayTrafficMark  = EnvOrUint32("XRAY_TRAFFIC_MARK", 0x1337)
)

func ParseApdCIDR() (*net.IPNet, error) {
	if ApdCIDR == "" {
		return nil, fmt.Errorf("APD_CIDR is not set")
	}

	_, cidr, err := net.ParseCIDR(ApdCIDR)
	if err != nil {
		return nil, fmt.Errorf("parse APD_CIDR %q: %w", ApdCIDR, err)
	}

	return cidr, nil
}

func TransmissionUID() (uint32, error) {
	u, err := user.Lookup(TransmissionUser)
	if err != nil {
		return 0, fmt.Errorf("lookup %q: %w", TransmissionUser, err)
	}

	uid, err := strconv.ParseUint(u.Uid, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("parse %q uid %q: %w", TransmissionUser, u.Uid, err)
	}

	return uint32(uid), nil
}

func CheckCap(cap int) error {
	hdr := unix.CapUserHeader{Version: unix.LINUX_CAPABILITY_VERSION_3}
	var data [2]unix.CapUserData

	if err := unix.Capget(&hdr, &data[0]); err != nil {
		return fmt.Errorf("unix.Capget: %v", err)
	}
	if data[cap/32].Effective&(1<<(cap%32)) == 0 && os.Getuid() != 0 {
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
