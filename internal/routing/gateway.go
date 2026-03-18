package routing

import (
	"fmt"
	"log"
	"net"

	"github.com/realglebivanov/xray-vpn/internal/routing/state"
	"github.com/vishvananda/netlink"
)

func preserveDefaultGateway() (*state.DefaultGateway, error) {
	gw, err := lookupDefaultGateway()
	if err == nil {
		if err := state.Save(gw); err != nil {
			return nil, fmt.Errorf("save default route: %w", err)
		}
		return gw, nil
	}

	log.Printf("no default gateway in routing table: %v; falling back to file", err)
	return state.Load()
}

func lookupDefaultGateway() (*state.DefaultGateway, error) {
	routes, err := netlink.RouteList(nil, netlink.FAMILY_V4)
	if err != nil {
		return nil, fmt.Errorf("list routes: %w", err)
	}
	for _, r := range routes {
		if r.Gw == nil || !isDefault(r) {
			continue
		}

		link, err := netlink.LinkByIndex(r.LinkIndex)
		if err != nil {
			continue
		}

		if link.Attrs().Name == TunDev {
			continue
		}

		return &state.DefaultGateway{Route: &r, IP: &r.Gw, Link: link}, nil
	}
	return nil, fmt.Errorf("no default gateway found")
}

func isDefault(r netlink.Route) bool {
	isDefault := r.Dst == nil
	if !isDefault && r.Dst != nil {
		ones, _ := r.Dst.Mask.Size()
		isDefault = r.Dst.IP.Equal(net.IPv4zero) && ones == 0
	}
	return isDefault
}
