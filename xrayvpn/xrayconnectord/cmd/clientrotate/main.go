package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/realglebivanov/hstd/hstdlib"
	"github.com/realglebivanov/hstd/hstdlib/secret"
)

type xrayConfig struct {
	Log       logConfig     `json:"log"`
	Inbounds  []inbound     `json:"inbounds"`
	Outbounds []outbound    `json:"outbounds"`
}

type logConfig struct {
	LogLevel string `json:"loglevel"`
	DNSLog   bool   `json:"dnsLog"`
}

type inbound struct {
	Listen         string         `json:"listen"`
	Port           int            `json:"port"`
	Protocol       string         `json:"protocol"`
	Settings       vlessSettings  `json:"settings"`
	StreamSettings streamSettings `json:"streamSettings"`
}

type vlessSettings struct {
	Clients    []vlessAccount `json:"clients"`
	Decryption string         `json:"decryption"`
}

type vlessAccount struct {
	ID   string `json:"id"`
	Flow string `json:"flow,omitempty"`
}

type streamSettings struct {
	Network         string           `json:"network"`
	Security        string           `json:"security"`
	TLSSettings     *tlsSettings     `json:"tlsSettings,omitempty"`
	REALITYSettings *realitySettings `json:"realitySettings,omitempty"`
	XHTTPSettings   *xhttpSettings   `json:"xhttpSettings,omitempty"`
}

type tlsSettings struct {
	Certificates []certificate `json:"certificates"`
}

type certificate struct {
	CertificateFile string `json:"certificateFile"`
	KeyFile         string `json:"keyFile"`
}

type realitySettings struct {
	Target      string   `json:"target"`
	ServerNames []string `json:"serverNames"`
	PrivateKey  string   `json:"privateKey"`
	ShortIds    []string `json:"shortIds"`
}

type xhttpSettings struct {
	Path              string `json:"path"`
	Mode              string `json:"mode"`
	XPaddingBytes     string `json:"xPaddingBytes,omitempty"`
	XPaddingObfsMode  bool   `json:"xPaddingObfsMode,omitempty"`
	XPaddingPlacement string `json:"xPaddingPlacement,omitempty"`
	XPaddingKey       string `json:"xPaddingKey,omitempty"`
	UplinkHTTPMethod  string `json:"uplinkHTTPMethod,omitempty"`
	NoGRPCHeader      bool   `json:"noGRPCHeader,omitempty"`
	NoSSEHeader       bool   `json:"noSSEHeader,omitempty"`
}

type outbound struct {
	Protocol string `json:"protocol"`
	Tag      string `json:"tag"`
}

func main() {
	if len(os.Args) < 2 || len(os.Args) > 3 {
		slog.Error("usage: clientrotate <secret> [flow]")
		os.Exit(1)
	}

	scrt, err := hstdlib.ParseHexSecret(os.Args[1])
	if err != nil {
		slog.Error("secret must be hex", "err", err)
		os.Exit(1)
	}

	var flow string
	if len(os.Args) == 3 {
		flow = os.Args[2]
	}

	uuids := secret.GenerateClientUUIDs(scrt)
	slog.Info("rotating client_id", "clients", len(uuids), "flow", flow)

	if err := rotate(uuids, flow); err != nil {
		slog.Error("rotate", "err", err)
		os.Exit(1)
	}
}

const configPath = "/usr/local/etc/xray/config.json"

func rotate(uuids []string, flow string) error {
	fi, err := os.Stat(configPath)
	if err != nil {
		return fmt.Errorf("stat config: %w", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	var cfg xrayConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}

	for i, inbound := range cfg.Inbounds {
		if inbound.Protocol != "vless" {
			continue
		}
		clients := make([]vlessAccount, len(uuids))
		for j, uuid := range uuids {
			clients[j] = vlessAccount{ID: uuid, Flow: flow}
		}
		cfg.Inbounds[i].Settings.Clients = clients
	}

	out, err := json.MarshalIndent(cfg, "", "    ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	out = append(out, '\n')

	if err := os.WriteFile(configPath, out, fi.Mode()); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	slog.Info("updated", "path", configPath)
	return nil
}
