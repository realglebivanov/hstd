package link

import (
	"fmt"
	"log/slog"
	"net"

	"github.com/realglebivanov/hstd/hstdlib"
	"github.com/vishvananda/netlink"
)

func preserveDefaultGateway() (*netlink.Route, error) {
	gw, link, err := lookupDefaultGateway()
	if err == nil {
		if err := save(gw, link); err != nil {
			return nil, fmt.Errorf("save default route: %w", err)
		}
		return gw, nil
	}

	slog.Warn("no default gateway in routing table, falling back to file", "err", err)
	return load()
}

func lookupDefaultGateway() (*netlink.Route, netlink.Link, error) {
	routes, err := netlink.RouteList(nil, netlink.FAMILY_V4)
	if err != nil {
		return nil, nil, fmt.Errorf("list routes: %w", err)
	}
	for _, r := range routes {
		link := getDefaultGatewayLink(&r)
		if link == nil {
			continue
		}

		if link.Attrs().Name == hstdlib.TunDev {
			continue
		}

		return &r, link, nil
	}
	return nil, nil, fmt.Errorf("no default gateway found")
}

func getDefaultGatewayLink(r *netlink.Route) netlink.Link {
	if r.Gw == nil || !isDefaultRoute(r) {
		return nil
	}

	link, err := netlink.LinkByIndex(r.LinkIndex)
	if err != nil {
		return nil
	}
	return link
}

func isDefaultRoute(r *netlink.Route) bool {
	isDefault := r.Dst == nil
	if !isDefault && r.Dst != nil {
		ones, _ := r.Dst.Mask.Size()
		isDefault = r.Dst.IP.Equal(net.IPv4zero) && ones == 0
	}
	return isDefault
}
