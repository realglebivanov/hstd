package config

import (
	"encoding/json"
	"fmt"

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

	outboundConfig, err := getActiveOutboundConfig()
	if err != nil {
		return nil, fmt.Errorf("active outbound config: %w", err)
	}

	tunJson, err := json.Marshal(map[string]any{"name": routing.TunDev, "mtu": routing.TunMTU})
	if err != nil {
		return nil, fmt.Errorf("tun setting json: %w", err)
	}

	tunSettings := json.RawMessage(tunJson)
	freedomSettings := json.RawMessage(`{"domainStrategy":"UseIP"}`)

	xrayCfg := buildCoreConfig(ruCIDRs, &tunSettings, &freedomSettings, outboundConfig)

	return xrayCfg.Build()
}

func buildCoreConfig(
	ruCIDRs []string,
	tunSettings *json.RawMessage,
	freedomSettings *json.RawMessage,
	outboundConfig *conf.OutboundDetourConfig,
) *conf.Config {
	return &conf.Config{
		LogConfig: &conf.LogConfig{
			AccessLog: "none",
			LogLevel:  "warning",
		},
		InboundConfigs: []conf.InboundDetourConfig{
			{
				Protocol: "tun",
				Tag:      "tun-in",
				Settings: tunSettings,
				SniffingConfig: &conf.SniffingConfig{
					Enabled:      true,
					DestOverride: conf.NewStringList([]string{"http", "tls", "quic"}),
				},
			},
		},
		OutboundConfigs: []conf.OutboundDetourConfig{
			{
				Protocol: "freedom",
				Tag:      "local",
				Settings: freedomSettings,
			},
			{
				Protocol: "freedom",
				Tag:      "direct",
				Settings: freedomSettings,
				StreamSetting: &conf.StreamConfig{
					SocketSettings: &conf.SocketConfig{
						Mark: int32(routing.Fwmark),
					},
				},
			},
			*outboundConfig,
		},
		RouterConfig: buildRouterConfig(outboundConfig.Tag, ruCIDRs),
		DNSConfig: &conf.DNSConfig{
			Servers: []*conf.NameServerConfig{
				{Address: &conf.Address{Address: net.ParseAddress("8.8.8.8")}},
				{Address: &conf.Address{Address: net.ParseAddress("8.8.4.4")}},
				{Address: &conf.Address{Address: net.ParseAddress("1.1.1.1")}},
			},
			QueryStrategy: "UseIP",
		},
	}
}

func buildRouterConfig(proxyTag string, ruCIDRs []string) *conf.RouterConfig {
	localRule, _ := json.Marshal(map[string]any{
		"type":        "field",
		"inboundTag":  "tun-in",
		"outboundTag": "local",
		"ip":          []string{"geoip:private"},
	})
	directRule, _ := json.Marshal(map[string]any{
		"type":        "field",
		"inboundTag":  "tun-in",
		"outboundTag": "direct",
		"ip":          append(ruCIDRs, "1.1.1.1", "8.8.8.8", "8.8.4.4", "geoip:ru"),
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
			localRule,
			directRule,
			fileTransferRule,
			proxyRule,
		},
		DomainStrategy: &domainStrategy,
	}
}
