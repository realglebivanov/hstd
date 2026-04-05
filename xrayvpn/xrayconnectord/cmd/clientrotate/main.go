package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/realglebivanov/hstd/hstdlib"
	"github.com/realglebivanov/hstd/hstdlib/secret"
	"github.com/realglebivanov/hstd/hstdlib/xrayconf"
)

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

	var cfg xrayconf.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}

	for i, inbound := range cfg.Inbounds {
		if inbound.Protocol != "vless" {
			continue
		}
		clients := make([]xrayconf.VLESSAccount, len(uuids))
		for j, uuid := range uuids {
			clients[j] = xrayconf.VLESSAccount{ID: uuid, Flow: flow}
		}
		cfg.Inbounds[i].Settings = xrayconf.VLESSInboundSettings{
			Clients:    clients,
			Decryption: "none",
		}
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
