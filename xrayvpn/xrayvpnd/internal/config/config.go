package config

import (
	"encoding/json"
	"fmt"

	"github.com/realglebivanov/hstd/hstdlib"
	"github.com/xtls/xray-core/common/net"
	core "github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/infra/conf"
)

func BuildCoreConfig(ruCIDRs []string) (*core.Config, error) {
	outboundConfig, err := getActiveOutboundConfig()
	if err != nil {
		return nil, fmt.Errorf("active outbound config: %w", err)
	}

	socksSettings := json.RawMessage(`{"auth":"noauth","udp":true, "ip": "127.0.0.1"}`)
	freedomSettings := json.RawMessage(`{"domainStrategy":"UseIP"}`)

	xrayCfg := &conf.Config{
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
				PortList: &conf.PortList{Range: []conf.PortRange{
					{From: hstdlib.SocksPort, To: hstdlib.SocksPort},
				}},
				Settings:       &socksSettings,
				SniffingConfig: &conf.SniffingConfig{Enabled: true},
			},
		},
		OutboundConfigs: []conf.OutboundDetourConfig{
			{
				Protocol: "freedom",
				Tag:      "direct",
				Settings: &freedomSettings,
				StreamSetting: &conf.StreamConfig{
					SocketSettings: &conf.SocketConfig{Mark: int32(hstdlib.XrayOutMark)},
				},
			},
			*outboundConfig,
		},
		RouterConfig: buildRouterConfig(outboundConfig.Tag, ruCIDRs),
		DNSConfig: &conf.DNSConfig{
			Servers: []*conf.NameServerConfig{
				{Address: &conf.Address{Address: net.ParseAddress("https+local://8.8.8.8/dns-query")}},
				{Address: &conf.Address{Address: net.ParseAddress("https+local://8.8.4.4/dns-query")}},
				{Address: &conf.Address{Address: net.ParseAddress("https+local://1.1.1.1/dns-query")}},
				{Address: &conf.Address{Address: net.ParseAddress("https+local://127.0.0.1/dns-query")}},
			},
			QueryStrategy: "UseIP",
		},
	}

	return xrayCfg.Build()
}

func buildRouterConfig(proxyTag string, ruCIDRs []string) *conf.RouterConfig {
	bittorrentRule, _ := json.Marshal(map[string]any{
		"type":        "field",
		"outboundTag": "direct",
		"protocol":    []string{"bittorrent"},
	})
	ftpRule, _ := json.Marshal(map[string]any{
		"type":        "field",
		"outboundTag": "direct",
		"port":        "20-21",
	})
	directRule, _ := json.Marshal(map[string]any{
		"type":        "field",
		"inboundTag":  "socks-in",
		"outboundTag": "direct",
		"ip":          append(ruCIDRs, "geoip:ru", "geoip:private"),
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
			bittorrentRule,
			ftpRule,
			directRule,
			dnsRule,
			proxyRule,
		},
		DomainStrategy: &domainStrategy,
	}
}
