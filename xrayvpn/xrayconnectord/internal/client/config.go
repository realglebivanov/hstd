package client

import "github.com/realglebivanov/hstd/hstdlib/xrayconf"

type ServerConfig struct {
	Remark     string `json:"remark"`
	Host       string `json:"host"`
	RealityPbk string `json:"realityPbk,omitempty"`
	RealitySni string `json:"realitySni,omitempty"`
	RealitySid string `json:"realitySid,omitempty"`
	XHTTPPath  string `json:"xhttpPath,omitempty"`
}

func BuildConfigs(clientID string, servers []*ServerConfig, routingRules []xrayconf.RouteRule) []xrayconf.Config {
	configs := make([]xrayconf.Config, len(servers))
	for i, srv := range servers {
		configs[i] = xrayconf.Config{
			Remarks:   srv.Remark,
			Log:       xrayconf.LogConfig{LogLevel: "warning"},
			DNS:       &xrayconf.DNSConfig{Servers: []string{"8.8.8.8", "1.1.1.1"}},
			Inbounds:  buildInbounds(),
			Outbounds: buildOutbounds(clientID, srv),
			Routing: &xrayconf.RoutingConfig{
				DomainStrategy: "IPIfNonMatch",
				Rules:          routingRules,
			},
		}
	}
	return configs
}

func buildInbounds() []xrayconf.Inbound {
	return []xrayconf.Inbound{
		{
			Tag:      "socks",
			Protocol: "socks",
			Port:     10808,
			Listen:   "127.0.0.1",
			Settings: &xrayconf.SocksSettings{UDP: true},
			Sniffing: &xrayconf.Sniffing{
				Enabled:      true,
				DestOverride: []string{"http", "tls", "quic"},
			},
		},
		{
			Tag:      "http",
			Protocol: "http",
			Port:     10809,
			Listen:   "127.0.0.1",
			Sniffing: &xrayconf.Sniffing{
				Enabled:      true,
				DestOverride: []string{"http", "tls", "quic"},
			},
		},
	}
}

func buildOutbounds(clientID string, srv *ServerConfig) []xrayconf.Outbound {
	return []xrayconf.Outbound{
		buildProxyOutbound(clientID, srv),
		{
			Tag:      "direct",
			Protocol: "freedom",
			Settings: xrayconf.FreedomSettings{DomainStrategy: "UseIP"},
		},
		{
			Tag:      "block",
			Protocol: "blackhole",
		},
	}
}

func buildProxyOutbound(clientID string, srv *ServerConfig) xrayconf.Outbound {
	if srv.XHTTPPath != "" {
		return buildXHTTPOutbound(clientID, srv)
	}
	return buildRealityOutbound(clientID, srv)
}

func buildRealityOutbound(clientID string, srv *ServerConfig) xrayconf.Outbound {
	return xrayconf.Outbound{
		Tag:      "proxy",
		Protocol: "vless",
		Settings: xrayconf.VLESSOutboundSettings{
			Vnext: []xrayconf.VLESSServer{{
				Address: srv.Host,
				Port:    443,
				Users: []xrayconf.VLESSAccount{{
					ID:         clientID,
					Flow:       "xtls-rprx-vision",
					Encryption: "none",
				}},
			}},
		},
		StreamSettings: &xrayconf.StreamConfig{
			Network:  "tcp",
			Security: "reality",
			REALITYSettings: &xrayconf.RealityConfig{
				Fingerprint: "chrome",
				ServerName:  srv.RealitySni,
				PublicKey:    srv.RealityPbk,
				PrivateKey:   srv.RealityPbk,
				ShortID:     srv.RealitySid,
			},
		},
	}
}

func buildXHTTPOutbound(clientID string, srv *ServerConfig) xrayconf.Outbound {
	return xrayconf.Outbound{
		Tag:      "proxy",
		Protocol: "vless",
		Settings: xrayconf.VLESSOutboundSettings{
			Vnext: []xrayconf.VLESSServer{{
				Address: srv.Host,
				Port:    443,
				Users: []xrayconf.VLESSAccount{{
					ID:         clientID,
					Encryption: "none",
				}},
			}},
		},
		StreamSettings: &xrayconf.StreamConfig{
			Network:  "xhttp",
			Security: "tls",
			TLSSettings: &xrayconf.TLSConfig{
				ServerName: srv.Host,
			},
			XHTTPSettings: &xrayconf.XHTTPConfig{
				Path:              srv.XHTTPPath,
				XPaddingBytes:     "10-100",
				XPaddingObfsMode:  true,
				XPaddingPlacement: "query",
				XPaddingKey:       "q",
				Mode:              "packet-up",
				UplinkHTTPMethod:  "PUT",
				NoGRPCHeader:      true,
				NoSSEHeader:       true,
			},
		},
	}
}
