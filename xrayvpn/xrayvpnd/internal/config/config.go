package config

import (
	"encoding/json"
	"fmt"

	"github.com/realglebivanov/hstd/hstdlib"
	"github.com/realglebivanov/hstd/hstdlib/xrayconf"
	"github.com/realglebivanov/hstd/xrayvpnd/internal/config/repo"
	core "github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/infra/conf"
)

func BuildCoreConfig(db *repo.DB) (*core.Config, error) {
	cfg, err := db.GetActiveConfig()
	if err != nil {
		return nil, fmt.Errorf("active config: %w", err)
	}

	cfg.Log = xrayconf.LogConfig{
		LogLevel: "warning",
		Access:   "none",
		DNSLog:   false,
	}
	cfg.DNS = &xrayconf.DNSConfig{
		Servers: []string{
			"https+local://8.8.8.8/dns-query",
			"https+local://8.8.4.4/dns-query",
			"https+local://1.1.1.1/dns-query",
			"https+local://127.0.0.1/dns-query",
		},
		QueryStrategy: "UseIP",
	}

	cfg.Inbounds = []*xrayconf.Inbound{
		{
			Tag:      "socks-in",
			Protocol: "socks",
			Listen:   hstdlib.SocksHost,
			Port:     hstdlib.SocksPort,
			Settings: &xrayconf.SocksSettings{
				Auth: "noauth",
				UDP:  true,
				IP:   "127.0.0.1",
			},
			Sniffing: &xrayconf.Sniffing{Enabled: true},
		},
	}

	for _, ob := range cfg.Outbounds {
		if ob.Tag == hstdlib.BlockTag {
			continue
		}

		if ob.StreamSettings == nil {
			ob.StreamSettings = &xrayconf.StreamConfig{}
		}
		if ob.StreamSettings.SocketSettings == nil {
			ob.StreamSettings.SocketSettings = &xrayconf.SocketConfig{}
		}
		ob.StreamSettings.SocketSettings.Mark = int32(hstdlib.XrayOutMark)
	}

	cfg.Routing = buildRouting(cfg.Routing)

	data, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("marshal config: %w", err)
	}

	var xrayCfg conf.Config
	if err := json.Unmarshal(data, &xrayCfg); err != nil {
		return nil, fmt.Errorf("unmarshal conf config: %w", err)
	}

	return xrayCfg.Build()
}

func buildRouting(cfg *xrayconf.RoutingConfig) *xrayconf.RoutingConfig {
	hstdRules := []xrayconf.RouteRule{
		{Type: "field", OutboundTag: hstdlib.DirectTag, Protocol: []string{"bittorrent"}},
		{Type: "field", OutboundTag: hstdlib.DirectTag, Port: "20-21"},
		{Type: "field", OutboundTag: hstdlib.DirectTag, IP: []string{"8.8.8.8", "8.8.4.4", "1.1.1.1"}},
	}

	var cfgRules []xrayconf.RouteRule
	if cfg != nil {
		cfgRules = cfg.Rules
	}

	return &xrayconf.RoutingConfig{
		DomainStrategy: "IPOnDemand",
		Rules:          append(hstdRules, cfgRules...),
	}
}
