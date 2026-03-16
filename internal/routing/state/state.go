package state

import (
	"encoding/json"
	"fmt"
	"net"
	"os"

	"github.com/vishvananda/netlink"
)

const gatewayFile = "/var/cache/xray-vpn/default-gateway.json"

type DefaultGateway struct {
	Route netlink.Route
	IP    net.IP
	Link  netlink.Link
}

type savedGateway struct {
	Route    netlink.Route `json:"route"`
	LinkName string        `json:"link_name"`
}

func Save(gw *DefaultGateway) error {
	sg := savedGateway{
		Route:    gw.Route,
		LinkName: gw.Link.Attrs().Name,
	}
	data, err := json.MarshalIndent(sg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal gateway: %w", err)
	}
	if err := os.WriteFile(gatewayFile, data, 0644); err != nil {
		return fmt.Errorf("write %s: %w", gatewayFile, err)
	}
	return nil
}

func Load() (*DefaultGateway, error) {
	data, err := os.ReadFile(gatewayFile)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", gatewayFile, err)
	}

	var sg savedGateway
	if err := json.Unmarshal(data, &sg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", gatewayFile, err)
	}

	link, err := netlink.LinkByName(sg.LinkName)
	if err != nil {
		return nil, fmt.Errorf("link %q: %w", sg.LinkName, err)
	}

	sg.Route.LinkIndex = link.Attrs().Index

	return &DefaultGateway{
		Route: sg.Route,
		IP:    sg.Route.Gw,
		Link:  link,
	}, nil
}
