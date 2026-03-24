package hstdlib

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"os"
	"os/user"
	"strconv"
	"time"

	"golang.org/x/sys/unix"
)

var (
	SocksHost = EnvOr("SOCKS_HOST", "127.0.0.1")
	SocksPort = EnvOrUint32("SOCKS_PORT", 1080)
	ApdCIDR   = EnvOr("APD_CIDR", "")

	XrayOutMark      = EnvOrUint32("XRAY_OUT_MARK", 0x1f)
	XrayTrafficMark  = EnvOrUint32("XRAY_TRAFFIC_MARK", 0x1337)
	DirectRouteTable = 100

	TunDev = EnvOr("TUN_DEV", "xray0")
	ApdDev = EnvOr("APD_DEV", "wlp4s0")
	WanDev = EnvOr("WAN_DEV", "eno1")
	TunMTU = EnvOrInt("TUN_MTU", 1500)

	XrayVpnPIDFile   = "/run/xrayvpn/xrayvpnd.pid"
	Tun2SocksPIDFile = "/run/xrayvpn/tun2socksd.pid"

	TransmissionUser    = EnvOr("TRANSMISSION_USER", "debian-transmission")
	NavidromeUser       = EnvOr("NAVIDROME_USER", "navidrome")
	DirectRouteServices = []string{TransmissionUser, NavidromeUser}
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

func LookupUID(username string) (uint32, error) {
	u, err := user.Lookup(username)
	if err != nil {
		return 0, fmt.Errorf("lookup %q: %w", username, err)
	}

	uid, err := strconv.ParseUint(u.Uid, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("parse %q uid %q: %w", username, u.Uid, err)
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

func MustEnvUint64(key string) uint64 {
	v := MustEnv(key)
	n, err := strconv.ParseUint(v, 10, 64)
	if err != nil {
		log.Fatalf("env var %s must be an integer: %v", key, err)
	}
	return n
}

func MustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("missing required env var %s", key)
	}
	return v
}

func GenerateClientUUID(secret uint64) string {
	now := time.Now().UTC()
	epoch := now.Add(-3 * time.Hour) // align day boundary with 03:00 UTC rotation schedule
	day := time.Date(epoch.Year(), epoch.Month(), epoch.Day(), 0, 0, 0, 0, time.UTC).Unix()
	h := sha256.Sum256(binary.BigEndian.AppendUint64(nil, uint64(day)+secret))
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		h[0:4], h[4:6], h[6:8], h[8:10], h[10:16])
}
