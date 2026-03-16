package routing

import (
	"errors"
	"fmt"
	"log"
	"syscall"

	"github.com/realglebivanov/xray-vpn/internal/routing/state"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

const (
	Fwmark           = 0x1f
	directRouteTable = 100
)

func populateRouteTable(tun *Tunnel) error {
	if err := netlink.RouteDel(&tun.Gw.Route); err != nil {
		return fmt.Errorf("default route delete %s: %w", tun.Gw.Route, err)
	}
	log.Printf("default direct route is down %s", tun.Gw.IP.String())

	if err := netlink.RuleAdd(buildFwmarkRule()); err != nil {
		return fmt.Errorf("add fwmark rule: %w", err)
	}
	if err := netlink.RouteAdd(buildDirectRoute(tun.Gw)); err != nil {
		return fmt.Errorf("add route: %w", err)
	}
	log.Printf("proxy direct route is up (fwmark %#x → table %d)", Fwmark, directRouteTable)

	defaultRoute := buildDefaultRoute(tun)
	if err := netlink.RouteReplace(defaultRoute); err != nil {
		return fmt.Errorf("route add %s: %w", defaultRoute.Gw.String(), err)
	}
	log.Printf("proxy default route is up %s", tun.TunAddr)

	return nil
}

func cleanRouteTable(tun *Tunnel) error {
	if err := netlink.RouteDel(buildDefaultRoute(tun)); err != nil && !errors.Is(err, syscall.ESRCH) {
		return fmt.Errorf("default route delete %s: %w", tun.Gw.Route, err)
	}
	if err := netlink.RouteReplace(&tun.Gw.Route); err != nil {
		return fmt.Errorf("default route replace %s: %w", tun.Gw.Route, err)
	}
	log.Printf("default route is restored %s", tun.Gw.IP.String())

	if err := netlink.RuleDel(buildFwmarkRule()); err != nil && !errors.Is(err, syscall.ENOENT) {
		return fmt.Errorf("delete fwmark rule: %w", err)
	}

	if err := netlink.RouteDel(buildDirectRoute(tun.Gw)); err != nil && !errors.Is(err, syscall.ESRCH) {
		return fmt.Errorf("delete route: %w", err)
	}
	log.Printf("direct route is deleted %s", tun.Gw.IP.String())

	return nil
}

func buildDefaultRoute(tun *Tunnel) *netlink.Route {
	return &netlink.Route{
		Dst:       nil,
		Gw:        tun.TunAddr.IP,
		LinkIndex: tun.TunLink.Attrs().Index,
		Table:     unix.RT_TABLE_MAIN,
		Priority:  0,
	}
}

func buildDirectRoute(gw *state.DefaultGateway) *netlink.Route {
	return &netlink.Route{
		Dst:       nil,
		Gw:        gw.IP,
		LinkIndex: gw.Link.Attrs().Index,
		Table:     directRouteTable,
		Priority:  0,
	}
}

func buildFwmarkRule() *netlink.Rule {
	rule := netlink.NewRule()
	rule.Mark = Fwmark
	rule.Table = directRouteTable
	return rule
}
