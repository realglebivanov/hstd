package link

import (
	"fmt"
	"log"
	"time"

	"github.com/realglebivanov/hstd/hstdlib"
	"github.com/vishvananda/netlink"
)

type Link struct {
	DefaultGw *netlink.Route
	TunLink   netlink.Link
	TunAddr   *netlink.Addr
}

func SetUp() (*Link, error) {
	defaultGw, err := preserveDefaultGateway()
	if err != nil {
		return nil, err
	}

	tunLink, err := await(10 * time.Second)
	if err != nil {
		return nil, err
	}

	tunAddr, err := configure(tunLink)
	if err != nil {
		return nil, err
	}

	return &Link{DefaultGw: defaultGw, TunLink: tunLink, TunAddr: tunAddr}, nil
}

func await(timeout time.Duration) (netlink.Link, error) {
	log.Printf("waiting for %s ...", hstdlib.TunDev)

	deadline := time.Now().Add(timeout)
	for {
		if link, err := netlink.LinkByName(hstdlib.TunDev); err == nil {
			return link, nil
		}
		if time.Now().After(deadline) {
			return nil, fmt.Errorf("timeout waiting for %s", hstdlib.TunDev)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

func configure(link netlink.Link) (*netlink.Addr, error) {
	addr, err := netlink.ParseAddr("198.18.0.1/16")
	if err != nil {
		return nil, err
	}
	if err := netlink.AddrReplace(link, addr); err != nil {
		return nil, fmt.Errorf("addr add on %s: %w", hstdlib.TunDev, err)
	}
	if err := netlink.LinkSetUp(link); err != nil {
		return nil, fmt.Errorf("link set up %s: %w", hstdlib.TunDev, err)
	}
	return addr, nil
}
