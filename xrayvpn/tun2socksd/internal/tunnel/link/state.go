package link

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/realglebivanov/hstd/hstdlib"
	"github.com/vishvananda/netlink"
)

var (
	cacheDir    = hstdlib.EnvOr("CACHE_DIRECTORY", "/var/cache/xrayvpn")
	gatewayFile = filepath.Join(cacheDir, "default-gateway.json")
)

type savedGateway struct {
	Route    *netlink.Route `json:"route"`
	LinkName string         `json:"link_name"`
}

func save(gw *netlink.Route, link netlink.Link) error {
	sg := savedGateway{Route: gw, LinkName: link.Attrs().Name}
	data, err := json.MarshalIndent(sg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal gateway: %w", err)
	}

	if err := os.WriteFile(gatewayFile, data, 0644); err != nil {
		return fmt.Errorf("write %s: %w", gatewayFile, err)
	}
	return nil
}

func load() (*netlink.Route, error) {
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

	return sg.Route, nil
}
