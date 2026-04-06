package xrayconf

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

type VLESSLink struct {
	UUID    string
	Address string
	Port    uint16

	Flow     string
	Network  string
	Security string

	SNI         string
	ALPN        string
	Fingerprint string
	PublicKey   string
	ShortID     string
	SpiderX     string

	Path        string
	Host        string
	ServiceName string
	HeaderType  string
	Seed        string
	Mode        string

	Fragment string
}

func ParseVLESSLink(raw string) (*VLESSLink, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return nil, fmt.Errorf("parse url: %w", err)
	}
	if u.Scheme != "vless" {
		return nil, fmt.Errorf("unsupported scheme %q", u.Scheme)
	}
	if u.Hostname() == "" {
		return nil, fmt.Errorf("missing hostname")
	}
	if u.Port() == "" {
		return nil, fmt.Errorf("missing port")
	}
	port, err := strconv.ParseUint(u.Port(), 10, 16)
	if err != nil {
		return nil, fmt.Errorf("invalid port: %w", err)
	}
	if u.User == nil {
		return nil, fmt.Errorf("missing uuid")
	}

	q := u.Query()

	return &VLESSLink{
		UUID:        u.User.Username(),
		Address:     u.Hostname(),
		Port:        uint16(port),
		Flow:        q.Get("flow"),
		Network:     or(q.Get("type"), "tcp"),
		Security:    or(q.Get("security"), "none"),
		SNI:         q.Get("sni"),
		ALPN:        q.Get("alpn"),
		Fingerprint: q.Get("fp"),
		PublicKey:   q.Get("pbk"),
		ShortID:     q.Get("sid"),
		SpiderX:     q.Get("spx"),
		Path:        q.Get("path"),
		Host:        q.Get("host"),
		ServiceName: q.Get("serviceName"),
		HeaderType:  q.Get("headerType"),
		Seed:        q.Get("seed"),
		Mode:        q.Get("mode"),
		Fragment:    u.Fragment,
	}, nil
}

func (l *VLESSLink) String() string {
	q := url.Values{}
	set := func(k, v string) {
		if v != "" {
			q.Set(k, v)
		}
	}
	set("type", l.Network)
	set("security", l.Security)
	set("flow", l.Flow)
	set("fp", l.Fingerprint)
	set("sni", l.SNI)
	set("alpn", l.ALPN)
	set("pbk", l.PublicKey)
	set("sid", l.ShortID)
	set("spx", l.SpiderX)
	set("path", l.Path)
	set("host", l.Host)
	set("serviceName", l.ServiceName)
	set("headerType", l.HeaderType)
	set("seed", l.Seed)
	set("mode", l.Mode)

	s := fmt.Sprintf("vless://%s@%s:%d?%s", l.UUID, l.Address, l.Port, q.Encode())
	if l.Fragment != "" {
		s += "#" + url.PathEscape(l.Fragment)
	}
	return s
}

func (l *VLESSLink) Outbound(tag string) Outbound {
	return Outbound{
		Tag:      tag,
		Protocol: "vless",
		Settings: &OutboundSettings{
			Vnext: []VLESSServer{{
				Address: l.Address,
				Port:    l.Port,
				Users: []VLESSAccount{{
					ID:         l.UUID,
					Flow:       l.Flow,
					Encryption: "none",
				}},
			}},
		},
		StreamSettings: l.streamConfig(),
	}
}

func (l *VLESSLink) streamConfig() *StreamConfig {
	sc := &StreamConfig{
		Network:  l.Network,
		Security: l.Security,
	}

	switch l.Network {
	case "tcp":
		if l.HeaderType == "http" {
			sc.TCPSettings = &TCPConfig{
				Header: map[string]any{
					"type": "http",
					"request": map[string]any{
						"path":    strings.Split(l.Path, ","),
						"headers": map[string][]string{"Host": strings.Split(l.Host, ",")},
					},
				},
			}
		}
	case "ws":
		sc.WSSettings = &WSConfig{
			Path:    l.Path,
			Headers: map[string]string{"Host": l.Host},
		}
	case "grpc":
		sc.GRPCSettings = &GRPCConfig{
			ServiceName: l.ServiceName,
			MultiMode:   l.Mode == "multi",
		}
	case "kcp":
		kcp := &KCPConfig{Seed: l.Seed}
		if l.HeaderType != "" {
			kcp.Header = map[string]string{"type": l.HeaderType}
		}
		sc.KCPSettings = kcp
	case "xhttp":
		sc.XHTTPSettings = &XHTTPConfig{
			Path:              l.Path,
			Mode:              l.Mode,
			XPaddingBytes:     "10-100",
			XPaddingObfsMode:  true,
			XPaddingPlacement: "query",
			XPaddingKey:       "q",
			UplinkHTTPMethod:  "PUT",
			NoGRPCHeader:      true,
			NoSSEHeader:       true,
		}
	}

	switch l.Security {
	case "tls":
		sc.TLSSettings = &TLSConfig{
			ServerName: l.SNI,
		}
	case "reality":
		sc.REALITYSettings = &RealityConfig{
			Fingerprint: l.Fingerprint,
			ServerName:  l.SNI,
			PublicKey:   l.PublicKey,
			ShortID:     l.ShortID,
		}
	}

	return sc
}

func (l *VLESSLink) SplitALPN() []string {
	if l.ALPN == "" {
		return nil
	}
	return strings.Split(l.ALPN, ",")
}

func or(val, fallback string) string {
	if val == "" {
		return fallback
	}
	return val
}
