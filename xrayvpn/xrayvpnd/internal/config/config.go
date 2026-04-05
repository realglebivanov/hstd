package config

import (
	"encoding/json"
	"fmt"

	"github.com/realglebivanov/hstd/hstdlib"
	"github.com/realglebivanov/hstd/hstdlib/xrayconf"
	"github.com/realglebivanov/hstd/xrayvpnd/internal/config/store"
	"github.com/xtls/xray-core/common/net"
	core "github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/infra/conf"
)

func BuildCoreConfig(ruCIDRs []string) (*core.Config, error) {
	outboundConfig, err := getActiveOutboundConfig()
	if err != nil {
		return nil, fmt.Errorf("active outbound config: %w", err)
	}

	routerConfig, err := buildRouterConfig(outboundConfig.Tag, ruCIDRs)
	if err != nil {
		return nil, fmt.Errorf("build router config: %w", err)
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
		RouterConfig: routerConfig,
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

func buildRouterConfig(proxyTag string, ruCIDRs []string) (*conf.RouterConfig, error) {
	rules := []xrayconf.RouteRule{
		{Type: "field", OutboundTag: "direct", Protocol: []string{"bittorrent"}},
		{Type: "field", OutboundTag: "direct", Port: "20-21"},
		{Type: "field", OutboundTag: "direct", InboundTag: "socks-in",
			IP: append(ruCIDRs, "geoip:ru", "geoip:private")},
		{Type: "field", OutboundTag: "direct", IP: []string{"8.8.8.8", "8.8.4.4", "1.1.1.1"}},
		{Type: "field", OutboundTag: proxyTag, Network: "tcp,udp"},
	}

	ruleList := make([]json.RawMessage, len(rules))
	for i, r := range rules {
		rule, err := json.Marshal(r)
		if err != nil {
			return nil, err
		}
		ruleList[i] = rule
	}

	domainStrategy := "IPOnDemand"
	return &conf.RouterConfig{
		RuleList:       ruleList,
		DomainStrategy: &domainStrategy,
	}, nil
}

func getActiveOutboundConfig() (*conf.OutboundDetourConfig, error) {
	rawLink, err := store.GetActiveLink()
	if err != nil {
		return nil, err
	}

	l, err := xrayconf.ParseVLESSLink(rawLink)
	if err != nil {
		return nil, err
	}

	ob := l.Outbound("proxy")
	ob.StreamSettings.SocketSettings = &xrayconf.SocketConfig{
		Mark: int32(hstdlib.XrayOutMark),
	}

	data, err := json.Marshal(ob)
	if err != nil {
		return nil, fmt.Errorf("marshal outbound: %w", err)
	}

	var out conf.OutboundDetourConfig
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, fmt.Errorf("unmarshal outbound: %w", err)
	}

	return &out, nil
}
