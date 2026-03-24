package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/realglebivanov/hstd/hstdlib"
	"github.com/xtls/xray-core/common/net"
	"github.com/xtls/xray-core/infra/conf"
	"github.com/xtls/xray-core/proxy/vless"
)

type namedConfig struct {
	Remarks string `json:"remarks"`
	*conf.Config
}

type xrayConfig struct {
	remark     string
	secret     uint64
	host       string
	realityPbk string
	realitySni string
	realitySid string
}

func main() {
	subPath := hstdlib.MustEnv("SUB_PATH")
	secret := hstdlib.MustEnvUint64("SECRET")
	xrayConfigs := []*xrayConfig{{
		remark:     "Direct",
		host:       hstdlib.MustEnv("SERVER_HOST"),
		realityPbk: hstdlib.MustEnv("REALITY_PBK"),
		realitySni: hstdlib.MustEnv("REALITY_SNI"),
		realitySid: hstdlib.MustEnv("REALITY_SID"),
	}, {
		remark:     "Proxy",
		host:       hstdlib.MustEnv("PROXY_HOST"),
		realityPbk: hstdlib.MustEnv("REALITY_PBK"),
		realitySni: hstdlib.MustEnv("REALITY_SNI"),
		realitySid: hstdlib.MustEnv("REALITY_SID"),
	}}

	http.HandleFunc("/"+subPath, func(w http.ResponseWriter, r *http.Request) {
		namedConfigs, err := buildNamedConfigs(secret, xrayConfigs)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("profile-update-interval", "1")
		if err := json.NewEncoder(w).Encode(namedConfigs); err != nil {
			log.Printf("encode response: %v", err)
		}
	})

	log.Println("listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func buildNamedConfigs(secret uint64, xrayConfigs []*xrayConfig) ([]namedConfig, error) {
	uuid := hstdlib.GenerateClientUUID(secret)

	var namedCfgs = make([]namedConfig, len(xrayConfigs))

	for i, xrayConfig := range xrayConfigs {
		proxyCfg, err := buildConfig(uuid, xrayConfig)
		if err != nil {
			return nil, fmt.Errorf("build proxy config: %v", err)
		}
		namedCfgs[i] = namedConfig{xrayConfig.remark, proxyCfg}
	}

	return namedCfgs, nil
}

func buildConfig(uuid string, xrayConfigs *xrayConfig) (*conf.Config, error) {
	outboundConfigs, err := buildOutbondConfigs(uuid, xrayConfigs)
	if err != nil {
		return nil, err
	}

	socksSettings, err := json.Marshal(map[string]any{"udp": true})
	if err != nil {
		return nil, err
	}

	ruleList, err := buildRuleList()
	if err != nil {
		return nil, err
	}

	domainStrategy := "IPIfNonMatch"
	socksRaw := json.RawMessage(socksSettings)

	return &conf.Config{
		LogConfig: &conf.LogConfig{
			LogLevel: "warning",
		},
		DNSConfig: &conf.DNSConfig{
			Servers: []*conf.NameServerConfig{
				{Address: &conf.Address{Address: net.ParseAddress("8.8.8.8")}},
				{Address: &conf.Address{Address: net.ParseAddress("1.1.1.1")}},
			},
		},
		InboundConfigs: []conf.InboundDetourConfig{
			{
				Tag:      "socks",
				Protocol: "socks",
				PortList: &conf.PortList{Range: []conf.PortRange{{From: 10808, To: 10808}}},
				ListenOn: &conf.Address{Address: net.ParseAddress("127.0.0.1")},
				Settings: &socksRaw,
				SniffingConfig: &conf.SniffingConfig{
					Enabled:      true,
					DestOverride: conf.NewStringList([]string{"http", "tls", "quic"}),
				},
			},
			{
				Tag:      "http",
				Protocol: "http",
				PortList: &conf.PortList{Range: []conf.PortRange{{From: 10809, To: 10809}}},
				ListenOn: &conf.Address{Address: net.ParseAddress("127.0.0.1")},
			},
		},
		OutboundConfigs: outboundConfigs,
		RouterConfig: &conf.RouterConfig{
			DomainStrategy: &domainStrategy,
			RuleList:       ruleList,
		},
	}, nil
}

func buildOutbondConfigs(uuid string, xrayConfig *xrayConfig) ([]conf.OutboundDetourConfig, error) {
	network := conf.TransportProtocol("tcp")

	freedomSettings, err := json.Marshal(map[string]any{"domainStrategy": "UseIP"})
	if err != nil {
		return nil, err
	}
	freedomRaw := json.RawMessage(freedomSettings)

	userJSON, err := json.Marshal(vless.Account{
		Id:         uuid,
		Flow:       "xtls-rprx-vision",
		Encryption: "none",
	})
	if err != nil {
		return nil, err
	}

	vlessSettings, err := json.Marshal(conf.VLessOutboundConfig{
		Vnext: []*conf.VLessOutboundVnext{{
			Address: &conf.Address{Address: net.ParseAddress(xrayConfig.host)},
			Port:    443,
			Users:   []json.RawMessage{userJSON},
		}},
	})
	vlessRaw := json.RawMessage(vlessSettings)

	return []conf.OutboundDetourConfig{
		{
			Tag:      "proxy",
			Protocol: "vless",
			Settings: &vlessRaw,
			StreamSetting: &conf.StreamConfig{
				Network:  &network,
				Security: "reality",
				REALITYSettings: &conf.REALITYConfig{
					Fingerprint: "chrome",
					ServerName:  xrayConfig.realitySni,
					PublicKey:   xrayConfig.realityPbk,
					ShortId:     xrayConfig.realitySid,
				},
			},
		},
		{
			Tag:      "direct",
			Protocol: "freedom",
			Settings: &freedomRaw,
		},
		{
			Tag:      "block",
			Protocol: "blackhole",
		},
	}, nil
}

func buildRuleList() ([]json.RawMessage, error) {
	rules := []map[string]any{
		{"type": "field", "outboundTag": "direct", "ip": []string{"geoip:ru", "geoip:private"}},
		{"type": "field", "outboundTag": "direct", "domain": []string{"geosite:category-ru", "geosite:category-gov-ru"}},
		{"type": "field", "outboundTag": "proxy", "network": "tcp,udp"},
	}

	ruleList := make([]json.RawMessage, len(rules))
	for i, r := range rules {
		b, err := json.Marshal(r)
		if err != nil {
			return nil, err
		}
		ruleList[i] = b
	}
	return ruleList, nil
}
