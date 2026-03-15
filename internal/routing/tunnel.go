package routing

import (
	"errors"
	"fmt"
	"log"
	"syscall"
	"time"

	"github.com/vishvananda/netlink"
)

type Tunnel struct {
	Gw      *DefaultGateway
	TunLink netlink.Link
	TunAddr *netlink.Addr
}

const (
	TunDev  = "xray0"
	TunMTU  = 1500
	TunAddr = "198.18.0.1"
)

func LinkExists(name string) bool {
	_, err := netlink.LinkByName(name)
	return err == nil
}

func TearDownTunnel(gw *Tunnel) error {
	if err := cleanRouteTable(gw); err != nil {
		return fmt.Errorf("clean table: %w", err)
	}
	if err := netlink.LinkDel(gw.TunLink); err != nil && !errors.Is(err, syscall.ENOENT) {
		return fmt.Errorf("delete link: %w", err)
	}
	return nil
}

func SetUpTunnel() (*Tunnel, error) {
	gw, err := detectDefaultGateway()
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
		return nil, fmt.Errorf("populate table: %w", err)
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
	addr, err := netlink.ParseAddr(fmt.Sprintf("%s/16", TunAddr))
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
