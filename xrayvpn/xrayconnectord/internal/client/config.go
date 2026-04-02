package client

type config struct {
	Remarks   string        `json:"remarks"`
	Log       logConfig     `json:"log"`
	DNS       dnsConfig     `json:"dns"`
	Inbounds  []inbound     `json:"inbounds"`
	Outbounds []outbound    `json:"outbounds"`
	Routing   routingConfig `json:"routing"`
}

type logConfig struct {
	LogLevel string `json:"loglevel"`
}

type dnsConfig struct {
	Servers []string `json:"servers"`
}

type inbound struct {
	Tag      string          `json:"tag"`
	Protocol string          `json:"protocol"`
	Port     uint16          `json:"port"`
	Listen   string          `json:"listen"`
	Settings *socksSettings  `json:"settings,omitempty"`
	Sniffing *sniffingConfig `json:"sniffing,omitempty"`
}

type socksSettings struct {
	UDP bool `json:"udp"`
}

type sniffingConfig struct {
	Enabled      bool     `json:"enabled"`
	DestOverride []string `json:"destOverride"`
}

type outbound struct {
	Tag            string        `json:"tag"`
	Protocol       string        `json:"protocol"`
	Settings       any           `json:"settings,omitempty"`
	StreamSettings *streamConfig `json:"streamSettings,omitempty"`
}

type vlessSettings struct {
	Vnext []vlessServer `json:"vnext"`
}

type vlessServer struct {
	Address string      `json:"address"`
	Port    uint16      `json:"port"`
	Users   []vlessUser `json:"users"`
}

type vlessUser struct {
	ID         string `json:"id"`
	Flow       string `json:"flow"`
	Encryption string `json:"encryption"`
}

type freedomSettings struct {
	DomainStrategy string `json:"domainStrategy"`
}

type streamConfig struct {
	Network         string         `json:"network"`
	Security        string         `json:"security"`
	REALITYSettings *realityConfig `json:"realitySettings,omitempty"`
	TLSSettings     *tlsConfig     `json:"tlsSettings,omitempty"`
	XHTTPSettings   *xhttpConfig   `json:"xhttpSettings,omitempty"`
}

type tlsConfig struct {
	ServerName string `json:"serverName"`
}

type xhttpConfig struct {
	Path              string `json:"path"`
	XPaddingBytes     string `json:"xPaddingBytes,omitempty"`
	XPaddingObfsMode  bool   `json:"xPaddingObfsMode,omitempty"`
	XPaddingPlacement string `json:"xPaddingPlacement,omitempty"`
	XPaddingKey       string `json:"xPaddingKey,omitempty"`
	Mode              string `json:"mode,omitempty"`
	UplinkHTTPMethod  string `json:"uplinkHTTPMethod,omitempty"`
	NoGRPCHeader      bool   `json:"noGRPCHeader,omitempty"`
	NoSSEHeader       bool   `json:"noSSEHeader,omitempty"`
}

type realityConfig struct {
	Fingerprint string   `json:"fingerprint"`
	ServerName  string   `json:"serverName"`
	ServerNames []string `json:"serverNames"`
	PublicKey   string   `json:"publicKey"`
	PrivateKey  string   `json:"privateKey"`
	ShortId     string   `json:"shortId"`
}

type routingConfig struct {
	DomainStrategy string      `json:"domainStrategy"`
	Rules          []routeRule `json:"rules"`
}

type routeRule struct {
	Type        string   `json:"type"`
	OutboundTag string   `json:"outboundTag"`
	IP          []string `json:"ip,omitempty"`
	Domain      []string `json:"domain,omitempty"`
	Network     string   `json:"network,omitempty"`
}

type ServerConfig struct {
	Remark     string
	Host       string
	RealityPbk string
	RealitySni string
	RealitySid string
	XHTTPPath  string
}

func BuildConfigs(clientID string, servers []*ServerConfig) []config {
	configs := make([]config, len(servers))
	for i, srv := range servers {
		configs[i] = config{
			Remarks:   srv.Remark,
			Log:       logConfig{LogLevel: "warning"},
			DNS:       dnsConfig{Servers: []string{"8.8.8.8", "1.1.1.1"}},
			Inbounds:  buildInbounds(),
			Outbounds: buildOutbounds(clientID, srv),
			Routing: routingConfig{
				DomainStrategy: "IPIfNonMatch",
				Rules: []routeRule{
					{Type: "field", OutboundTag: "proxy", Domain: []string{"domain:yonote.ru"}},
					{Type: "field", OutboundTag: "direct", IP: []string{"geoip:ru", "geoip:private"}},
					{Type: "field", OutboundTag: "direct", Domain: []string{"geosite:category-ru", "geosite:category-gov-ru"}},
					{Type: "field", OutboundTag: "proxy", Network: "tcp,udp"},
				},
			},
		}
	}
	return configs
}

func buildInbounds() []inbound {
	return []inbound{
		{
			Tag:      "socks",
			Protocol: "socks",
			Port:     10808,
			Listen:   "127.0.0.1",
			Settings: &socksSettings{UDP: true},
			Sniffing: &sniffingConfig{
				Enabled:      true,
				DestOverride: []string{"http", "tls", "quic"},
			},
		},
		{
			Tag:      "http",
			Protocol: "http",
			Port:     10809,
			Listen:   "127.0.0.1",
			Sniffing: &sniffingConfig{
				Enabled:      true,
				DestOverride: []string{"http", "tls", "quic"},
			},
		},
	}
}

func buildOutbounds(clientID string, srv *ServerConfig) []outbound {
	return []outbound{
		buildProxyOutbound(clientID, srv),
		{
			Tag:      "direct",
			Protocol: "freedom",
			Settings: freedomSettings{DomainStrategy: "UseIP"},
		},
		{
			Tag:      "block",
			Protocol: "blackhole",
		},
	}
}

func buildProxyOutbound(clientID string, srv *ServerConfig) outbound {
	if srv.XHTTPPath != "" {
		return buildXHTTPOutbound(clientID, srv)
	}
	return buildRealityOutbound(clientID, srv)
}

func buildRealityOutbound(clientID string, srv *ServerConfig) outbound {
	return outbound{
		Tag:      "proxy",
		Protocol: "vless",
		Settings: vlessSettings{
			Vnext: []vlessServer{{
				Address: srv.Host,
				Port:    443,
				Users: []vlessUser{{
					ID:         clientID,
					Flow:       "xtls-rprx-vision",
					Encryption: "none",
				}},
			}},
		},
		StreamSettings: &streamConfig{
			Network:  "tcp",
			Security: "reality",
			REALITYSettings: &realityConfig{
				Fingerprint: "chrome",
				ServerName:  srv.RealitySni,
				PublicKey:   srv.RealityPbk,
				PrivateKey:  srv.RealityPbk,
				ShortId:     srv.RealitySid,
			},
		},
	}
}

func buildXHTTPOutbound(clientID string, srv *ServerConfig) outbound {
	return outbound{
		Tag:      "proxy",
		Protocol: "vless",
		Settings: vlessSettings{
			Vnext: []vlessServer{{
				Address: srv.Host,
				Port:    443,
				Users: []vlessUser{{
					ID:         clientID,
					Encryption: "none",
				}},
			}},
		},
		StreamSettings: &streamConfig{
			Network:  "xhttp",
			Security: "tls",
			TLSSettings: &tlsConfig{
				ServerName: srv.Host,
			},
			XHTTPSettings: &xhttpConfig{
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
