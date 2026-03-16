package store

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/xtls/xray-core/infra/conf"
)

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
		Tag:           u.Fragment,
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
		Network:  (*conf.TransportProtocol)(&network),
		Security: security,
	}

	switch network {
	case "tcp":
		headerType := q.Get("headerType")
		if headerType == "http" {
			h := map[string]any{
				"header": map[string]any{
					"type": headerType,
					"request": map[string]any{
						"path":    splitCSV(q.Get("path")),
						"headers": map[string][]string{"Host": splitCSV(q.Get("host"))},
					},
				},
			}
			raw, _ := json.Marshal(h)
			sc.TCPSettings = &conf.TCPConfig{}
			json.Unmarshal(raw, sc.TCPSettings)
		}
	case "ws":
		raw, _ := json.Marshal(map[string]any{
			"path":    q.Get("path"),
			"headers": map[string]string{"Host": q.Get("host")},
		})
		sc.WSSettings = &conf.WebSocketConfig{}
		json.Unmarshal(raw, sc.WSSettings)
	case "grpc":
		raw, _ := json.Marshal(map[string]any{
			"serviceName": q.Get("serviceName"),
			"multiMode":   q.Get("mode") == "multi",
		})
		sc.GRPCSettings = &conf.GRPCConfig{}
		json.Unmarshal(raw, sc.GRPCSettings)
	case "kcp":
		m := map[string]any{"seed": q.Get("seed")}
		if ht := q.Get("headerType"); ht != "" {
			m["header"] = map[string]string{"type": ht}
		}
		raw, _ := json.Marshal(m)
		sc.KCPSettings = &conf.KCPConfig{}
		json.Unmarshal(raw, sc.KCPSettings)
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
