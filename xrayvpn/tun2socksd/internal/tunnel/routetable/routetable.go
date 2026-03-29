package routetable

import (
	"errors"
	"fmt"
	"log/slog"
	"syscall"

	"github.com/realglebivanov/hstd/hstdlib"
	"github.com/realglebivanov/hstd/tun2socksd/internal/tunnel/link"
	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

type routeOp int

const (
	del routeOp = iota
	replace
)

type routeStep struct {
	op    routeOp
	route *netlink.Route
	desc  string
}

type RouteTable struct {
	setUp    []routeStep
	tearDown []routeStep
}

func (rt RouteTable) SetUp() error {
	return execRouteSteps(rt.setUp)
}

func (rt RouteTable) TearDown() error {
	return execRouteSteps(rt.tearDown)
}

func New(link *link.Link) *RouteTable {
	return &RouteTable{
		setUp: []routeStep{
			{del, link.DefaultGw, "default route down"},
			{replace, buildDirectRoute(link.DefaultGw), "proxy direct route up"},
			{replace, buildDefaultRoute(link.TunAddr, link.TunLink), "proxy default route up"},
		},
		tearDown: []routeStep{
			{del, buildDefaultRoute(link.TunAddr, link.TunLink), "proxy default route down"},
			{replace, link.DefaultGw, "default route restored"},
			{del, buildDirectRoute(link.DefaultGw), "proxy direct route deleted"},
		},
	}
}

func execRouteSteps(steps []routeStep) error {
	for _, s := range steps {
		err := execRouteStep(&s)
		if err != nil && !errors.Is(err, syscall.ESRCH) {
			return fmt.Errorf("%s: %w", s.desc, err)
		}
		slog.Info(s.desc)
	}
	return nil
}

func execRouteStep(s *routeStep) error {
	switch s.op {
	case del:
		return netlink.RouteDel(s.route)
	case replace:
		return netlink.RouteReplace(s.route)
	}
	return nil
}

func buildDefaultRoute(tunAddr *netlink.Addr, tunLink netlink.Link) *netlink.Route {
	return &netlink.Route{
		Dst:       nil,
		Gw:        tunAddr.IP,
		LinkIndex: tunLink.Attrs().Index,
		Table:     unix.RT_TABLE_MAIN,
		Priority:  0,
	}
}

func buildDirectRoute(defaultGw *netlink.Route) *netlink.Route {
	return &netlink.Route{
		Dst:       nil,
		Gw:        defaultGw.Gw,
		LinkIndex: defaultGw.LinkIndex,
		Table:     hstdlib.DirectRouteTable,
		Priority:  0,
	}
}
