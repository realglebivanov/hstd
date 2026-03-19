package config

import (
	"encoding/json"
	"fmt"

	"github.com/realglebivanov/hstd/hstdlib"
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

	socksSettings := json.RawMessage(`{"auth":"noauth","udp":true, "ip": "127.0.0.1"}`)
	freedomSettings := json.RawMessage(`{"domainStrategy":"UseIP"}`)

	xrayCfg := buildCoreConfig(ruCIDRs, &socksSettings, &freedomSettings, outboundConfig)

	return xrayCfg.Build()
}

func buildCoreConfig(
	ruCIDRs []string,
	socksSettings *json.RawMessage,
	freedomSettings *json.RawMessage,
	outboundConfig *conf.OutboundDetourConfig,
) *conf.Config {
	return &conf.Config{
		LogConfig: &conf.LogConfig{
			AccessLog: "none",
			LogLevel:  "warning",
			DNSLog:    false,
		},
		InboundConfigs: []conf.InboundDetourConfig{
			{
				Protocol: "socks",
				Tag:      "socks-in",
				ListenOn: &conf.Address{Address: net.ParseAddress(hstdlib.SocksHost)},
				PortList: &conf.PortList{Range: []conf.PortRange{{From: hstdlib.SocksPort, To: hstdlib.SocksPort}}},
				Settings: socksSettings,
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
					SocketSettings: &conf.SocketConfig{Mark: int32(hstdlib.Fwmark)},
				},
			},
			*outboundConfig,
		},
		RouterConfig: buildRouterConfig(outboundConfig.Tag, ruCIDRs),
		DNSConfig: &conf.DNSConfig{
			Servers: []*conf.NameServerConfig{
				{Address: &conf.Address{Address: net.ParseAddress("https+local://1.1.1.1/dns-query")}},
			},
			QueryStrategy: "UseIP",
		},
	}
}

func buildRouterConfig(proxyTag string, ruCIDRs []string) *conf.RouterConfig {
	localRule, _ := json.Marshal(map[string]any{
		"type":        "field",
		"inboundTag":  "socks-in",
		"outboundTag": "local",
		"ip":          []string{"geoip:private"},
	})
	directRule, _ := json.Marshal(map[string]any{
		"type":        "field",
		"inboundTag":  "socks-in",
		"outboundTag": "direct",
		"ip":          append(ruCIDRs, "geoip:ru"),
	})
	fileTransferRule, _ := json.Marshal(map[string]any{
		"type":        "field",
		"inboundTag":  "socks-in",
		"outboundTag": "direct",
		"protocol":    []string{"bittorrent", "ftp"},
	})
	dnsRule, _ := json.Marshal(map[string]any{
		"type":        "field",
		"outboundTag": "direct",
		"ip":          []string{"8.8.8.8", "8.8.4.4", "1.1.1.1"},
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
			dnsRule,
			proxyRule,
		},
		DomainStrategy: &domainStrategy,
	}
}
