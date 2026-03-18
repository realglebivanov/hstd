package routing

import (
	"errors"
	"fmt"
	"log"
	"syscall"
	"time"

	"github.com/realglebivanov/xray-vpn/internal/routing/state"
	"github.com/realglebivanov/xray-vpn/internal/std"
	"github.com/vishvananda/netlink"
)

type Tunnel struct {
	Gw      *state.DefaultGateway
	TunLink netlink.Link
	TunAddr *netlink.Addr
}

var (
	TunDev = std.EnvOr("TUN_DEV", "xray0")
	LanDev = std.EnvOr("LAN_DEV", "wlp4s0")
	WanDev = std.EnvOr("WAN_DEV", "eno1")
	TunMTU = std.EnvOrInt("TUN_MTU", 1500)
)

func TearDownTunnel(tun *Tunnel) error {
	if err := removeForwardToTun(); err != nil {
		return fmt.Errorf("firewall: %w", err)
	}
	if err := cleanRouteTable(tun); err != nil {
		return fmt.Errorf("clean table: %w", err)
	}
	if err := netlink.LinkDel(tun.TunLink); err != nil && !errors.Is(err, syscall.ENOENT) {
		return fmt.Errorf("delete link: %w", err)
	}
	return nil
}

func SetUpTunnel() (*Tunnel, error) {
	gw, err := preserveDefaultGateway()
	if err != nil {
		return nil, err
	}

	tunLink, err := awaitTunLink(10 * time.Second)
	if err != nil {
		return nil, err
	}

	tunAddr, err := configureTun(tunLink)
	if err != nil {
		return nil, err
	}
	tunnel := Tunnel{Gw: gw, TunLink: tunLink, TunAddr: tunAddr}

	if err := populateRouteTable(&tunnel); err != nil {
		return &tunnel, fmt.Errorf("populate table: %w", err)
	}
	if err := allowForwardToTun(); err != nil {
		return &tunnel, fmt.Errorf("firewall: %w", err)
	}
	return &tunnel, nil
}

func awaitTunLink(timeout time.Duration) (netlink.Link, error) {
	log.Printf("waiting for %s ...", TunDev)

	deadline := time.Now().Add(timeout)
	for {
		if link, err := netlink.LinkByName(TunDev); err == nil {
			return link, nil
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timeout waiting for %s", TunDev)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func configureTun(link netlink.Link) (*netlink.Addr, error) {
	addr, err := netlink.ParseAddr("198.18.0.1/16")
	if err != nil {
		return nil, err
	}
	if err := netlink.AddrReplace(link, addr); err != nil {
		return nil, fmt.Errorf("addr add on %s: %w", TunDev, err)
	}
	if err := netlink.LinkSetUp(link); err != nil {
		return nil, fmt.Errorf("link set up %s: %w", TunDev, err)
	}
	return addr, nil
}
