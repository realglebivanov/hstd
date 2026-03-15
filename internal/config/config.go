package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/realglebivanov/xray-vpn/internal/routing"
	"github.com/xtls/xray-core/common/net"
	core "github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/infra/conf"
)

const (
	outboundConfigPath = "/etc/xray-vpn/config.json"
	ruCIDRsURL         = "https://raw.githubusercontent.com/ipverse/rir-ip/master/country/ru/ipv4-aggregated.txt"
)

func BuildCoreConfig() (*core.Config, error) {
	ruCIDRs, err := fetchRuCIDRs()
	if err != nil {
		return nil, err
	}

	proxyOut, err := buildProxyOutbound()
	if err != nil {
		return nil, err
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
				&conf.NameServerConfig{
					Address: &conf.Address{
						Address: net.ParseAddress("8.8.8.8"),
					},
				},
				&conf.NameServerConfig{
					Address: &conf.Address{
						Address: net.ParseAddress("8.8.4.4"),
					},
				},
				&conf.NameServerConfig{
					Address: &conf.Address{
						Address: net.ParseAddress("1.1.1.1"),
					},
				},
			},
			QueryStrategy: "UseIP",
		},
	}

	return xrayCfg.Build()
}

func buildProxyOutbound() (*conf.OutboundDetourConfig, error) {
	raw, err := os.ReadFile(outboundConfigPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", outboundConfigPath, err)
	}

	var out conf.OutboundDetourConfig
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("parse %s as outbound: %w", outboundConfigPath, err)
	}

	if out.Protocol == "" {
		return nil, fmt.Errorf("%s: missing \"protocol\" field", outboundConfigPath)
	}

	injectMark(&out)
	if out.Tag == "" {
		out.Tag = "proxy"
	}

	return &out, nil
}

func injectMark(out *conf.OutboundDetourConfig) {
	if out.StreamSetting == nil {
		out.StreamSetting = &conf.StreamConfig{}
	}
	if out.StreamSetting.SocketSettings == nil {
		out.StreamSetting.SocketSettings = &conf.SocketConfig{}
	}
	out.StreamSetting.SocketSettings.Mark = int32(routing.Fwmark)
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
