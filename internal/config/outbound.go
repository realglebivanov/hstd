package config

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/realglebivanov/xray-vpn/internal/config/store"
	"github.com/realglebivanov/xray-vpn/internal/routing"
	"github.com/xtls/xray-core/infra/conf"
)

func getActiveOutboundConfig() (*conf.OutboundDetourConfig, error) {
	st, err := store.GetState()
	if err != nil {
		return nil, err
	}

	if st.ActiveID == "" {
		return nil, fmt.Errorf("no active link selected")
	}

	for _, item := range st.Links {
		if item.ID == st.ActiveID {
			return parseLink(item.Link)
		}
	}

	return nil, fmt.Errorf("active link %q not found in state", st.ActiveID)
}

func parseLink(rawLink string) (*conf.OutboundDetourConfig, error) {
	u, err := url.Parse(rawLink)
	if err != nil {
		return nil, fmt.Errorf("parse url: %w", err)
	}
	if u.Scheme != "vless" {
		return nil, fmt.Errorf("unsupported scheme %q, only vless is supported", u.Scheme)
	}

	port, err := strconv.Atoi(u.Port())
	if err != nil {
		return nil, fmt.Errorf("invalid port: %w", err)
	}

	q := u.Query()

	userJSON, _ := json.Marshal(map[string]string{
		"id":         u.User.Username(),
		"flow":       q.Get("flow"),
		"encryption": "none",
	})
	settings, _ := json.Marshal(map[string]any{
		"vnext": []map[string]any{
			{
				"address": u.Hostname(),
				"port":    port,
				"users":   []json.RawMessage{userJSON},
			},
		},
	})
	raw := json.RawMessage(settings)

	stream := buildStreamSettings(q)

	return &conf.OutboundDetourConfig{
		Protocol:      "vless",
		Tag:           "proxy",
		Settings:      &raw,
		StreamSetting: stream,
	}, nil
}

func buildStreamSettings(q url.Values) *conf.StreamConfig {
	network := q.Get("type")
	if network == "" {
		network = "tcp"
	}
	security := q.Get("security")
	if security == "" {
		security = "none"
	}

	sc := &conf.StreamConfig{
		Network:        (*conf.TransportProtocol)(&network),
		Security:       security,
		SocketSettings: &conf.SocketConfig{Mark: int32(routing.Fwmark)},
	}

	switch network {
	case "tcp":
		if q.Get("headerType") == "http" {
			raw, _ := json.Marshal(map[string]any{
				"type": "http",
				"request": map[string]any{
					"path":    splitCSV(q.Get("path")),
					"headers": map[string][]string{"Host": splitCSV(q.Get("host"))},
				},
			})
			sc.TCPSettings = &conf.TCPConfig{HeaderConfig: raw}
		}
	case "ws":
		sc.WSSettings = &conf.WebSocketConfig{
			Path:    q.Get("path"),
			Headers: map[string]string{"Host": q.Get("host")},
		}
	case "grpc":
		sc.GRPCSettings = &conf.GRPCConfig{
			ServiceName: q.Get("serviceName"),
			MultiMode:   q.Get("mode") == "multi",
		}
	case "kcp":
		seed := q.Get("seed")
		kcp := &conf.KCPConfig{Seed: &seed}
		if ht := q.Get("headerType"); ht != "" {
			raw, _ := json.Marshal(map[string]string{"type": ht})
			kcp.HeaderConfig = raw
		}
		sc.KCPSettings = kcp
	}

	switch security {
	case "tls":
		sc.TLSSettings = &conf.TLSConfig{
			ServerName:  q.Get("sni"),
			Fingerprint: q.Get("fp"),
		}
		if alpn := q.Get("alpn"); alpn != "" {
			sc.TLSSettings.ALPN = conf.NewStringList(splitCSV(alpn))
		}
	case "reality":
		sc.REALITYSettings = &conf.REALITYConfig{
			Fingerprint: q.Get("fp"),
			ServerName:  q.Get("sni"),
			PublicKey:   q.Get("pbk"),
			ShortId:     q.Get("sid"),
			SpiderX:     q.Get("spx"),
		}
	}

	return sc
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, ",")
}
