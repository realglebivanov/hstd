package main

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"path/filepath"

	"github.com/realglebivanov/hstd/hstdlib"
)

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

type serverConfig struct {
	remark     string
	host       string
	realityPbk string
	realitySni string
	realitySid string
}

func main() {
	subPath := hstdlib.MustEnv("SUB_PATH")
	secret := hstdlib.MustEnvUint64("SECRET")

	servers := []*serverConfig{{
		remark:     "Обычный ВПН",
		host:       hstdlib.MustEnv("SERVER_HOST"),
		realityPbk: hstdlib.MustEnv("REALITY_PBK"),
		realitySni: hstdlib.MustEnv("REALITY_SNI"),
		realitySid: hstdlib.MustEnv("REALITY_SID"),
	}, {
		remark:     "Обход белых списков",
		host:       hstdlib.MustEnv("PROXY_HOST"),
		realityPbk: hstdlib.MustEnv("REALITY_PBK"),
		realitySni: hstdlib.MustEnv("REALITY_SNI"),
		realitySid: hstdlib.MustEnv("REALITY_SID"),
	}}

	http.HandleFunc("/"+subPath, func(w http.ResponseWriter, r *http.Request) {
		uuid := hstdlib.GenerateClientUUID(secret)
		configs := buildConfigs(uuid, servers)

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("profile-update-interval", "1")
		w.Header().Set("profile-title", "base64:"+base64.StdEncoding.EncodeToString([]byte("hstd")))

		if err := json.NewEncoder(w).Encode(configs); err != nil {
			log.Printf("encode response: %v", err)
		}
	})

	credsDir := hstdlib.MustEnv("CREDENTIALS_DIRECTORY")
	certFile := filepath.Join(credsDir, "tls_cert")
	keyFile := filepath.Join(credsDir, "tls_key")
	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServeTLS(":8080", certFile, keyFile, nil))
}

func buildConfigs(uuid string, servers []*serverConfig) []config {
	configs := make([]config, len(servers))
	for i, srv := range servers {
		configs[i] = config{
			Remarks:   srv.remark,
			Log:       logConfig{LogLevel: "warning"},
			DNS:       dnsConfig{Servers: []string{"8.8.8.8", "1.1.1.1"}},
			Inbounds:  buildInbounds(),
			Outbounds: buildOutbounds(uuid, srv),
			Routing: routingConfig{
				DomainStrategy: "IPIfNonMatch",
				Rules: []routeRule{
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
		},
	}
}

func buildOutbounds(uuid string, srv *serverConfig) []outbound {
	return []outbound{
		{
			Tag:      "proxy",
			Protocol: "vless",
			Settings: vlessSettings{
				Vnext: []vlessServer{{
					Address: srv.host,
					Port:    443,
					Users: []vlessUser{{
						ID:         uuid,
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
					ServerName:  srv.realitySni,
					PublicKey:   srv.realityPbk,
					PrivateKey:  srv.realityPbk,
					ShortId:     srv.realitySid,
				},
			},
		},
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
