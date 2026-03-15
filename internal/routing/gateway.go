package routing

import (
	"fmt"
	"net"

	"github.com/vishvananda/netlink"
)

type DefaultGateway struct {
	Route netlink.Route
	IP    net.IP
	Link  netlink.Link
	Iface string
}

func detectDefaultGateway() (*DefaultGateway, error) {
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

		if link.Attrs().Alias == TunDev {
			continue
		}

		return &DefaultGateway{
			Route: r,
			IP:    r.Gw,
			Link:  link,
			Iface: link.Attrs().Name,
		}, nil
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
