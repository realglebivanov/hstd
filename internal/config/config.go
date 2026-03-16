package config

import (
	"encoding/json"
	"fmt"

	"github.com/realglebivanov/xray-vpn/internal/config/store"
	"github.com/realglebivanov/xray-vpn/internal/routing"
	"github.com/xtls/xray-core/common/net"
	core "github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/infra/conf"
)

func BuildCoreConfig() (*core.Config, error) {
	if err := loadGeodata(); err != nil {
		return nil, fmt.Errorf("load geodata: %w", err)
	}

	ruCIDRs, err := loadRuCIDRs()
	if err != nil {
		return nil, fmt.Errorf("load ru CIDRS: %w", err)
	}

	proxyOut, err := buildProxyOutbound()
	if err != nil {
		return nil, fmt.Errorf("build proxy outbound: %w", err)
	}

	xrayCfg := &conf.Config{
		LogConfig: &conf.LogConfig{
			AccessLog: "none",
			LogLevel:  "warning",
		},
		InboundConfigs: []conf.InboundDetourConfig{
			buildTunInbound(),
		},
		OutboundConfigs: []conf.OutboundDetourConfig{
			buildDirectOutbound(routing.Fwmark),
			*proxyOut,
		},
		RouterConfig: buildRouterConfig(proxyOut.Tag, ruCIDRs),
		DNSConfig: &conf.DNSConfig{
			Servers: []*conf.NameServerConfig{
				{Address: &conf.Address{Address: net.ParseAddress("8.8.8.8")}},
				{Address: &conf.Address{Address: net.ParseAddress("8.8.4.4")}},
				{Address: &conf.Address{Address: net.ParseAddress("1.1.1.1")}},
			},
			QueryStrategy: "UseIP",
		},
	}

	return xrayCfg.Build()
}

func buildProxyOutbound() (*conf.OutboundDetourConfig, error) {
	out, err := store.GetActiveOutboundConfig()
	if err != nil {
		return nil, err
	}

	if out.StreamSetting == nil {
		out.StreamSetting = &conf.StreamConfig{}
	}
	if out.StreamSetting.SocketSettings == nil {
		out.StreamSetting.SocketSettings = &conf.SocketConfig{}
	}
	out.StreamSetting.SocketSettings.Mark = int32(routing.Fwmark)

	if out.Tag == "" {
		out.Tag = "proxy"
	}

	return out, nil
}

func buildDirectOutbound(mark int) conf.OutboundDetourConfig {
	freedomSettings := json.RawMessage(`{"domainStrategy":"UseIP"}`)
	out := conf.OutboundDetourConfig{
		Protocol: "freedom",
		Tag:      "direct",
		Settings: &freedomSettings,
		StreamSetting: &conf.StreamConfig{
			SocketSettings: &conf.SocketConfig{
				Mark: int32(mark),
			},
		},
	}
	return out
}

func buildTunInbound() conf.InboundDetourConfig {
	tunJson, _ := json.Marshal(map[string]any{
		"name": routing.TunDev,
		"mtu":  routing.TunMTU,
	})
	tunSettings := json.RawMessage(tunJson)

	return conf.InboundDetourConfig{
		Protocol: "tun",
		Tag:      "tun-in",
		Settings: &tunSettings,
		SniffingConfig: &conf.SniffingConfig{
			Enabled:      true,
			DestOverride: conf.NewStringList([]string{"http", "tls", "quic"}),
		},
	}
}

func buildRouterConfig(proxyTag string, ruCIDRs []string) *conf.RouterConfig {
	directIPs := []string{"geoip:private", "geoip:ru"}
	directIPs = append(directIPs, ruCIDRs...)

	ipRule, _ := json.Marshal(map[string]any{
		"type":        "field",
		"inboundTag":  "tun-in",
		"outboundTag": "direct",
		"ip":          directIPs,
	})

	fileTransferRule, _ := json.Marshal(map[string]any{
		"type":        "field",
		"inboundTag":  "tun-in",
		"outboundTag": "direct",
		"protocol":    []string{"bittorrent", "ftp"},
	})

	proxyRule, _ := json.Marshal(map[string]any{
		"type":        "field",
		"network":     "tcp,udp",
		"outboundTag": proxyTag,
	})

	domainStrategy := "IPOnDemand"

	return &conf.RouterConfig{
		RuleList: []json.RawMessage{
			ipRule,
			fileTransferRule,
			proxyRule,
		},
		DomainStrategy: &domainStrategy,
	}
}
