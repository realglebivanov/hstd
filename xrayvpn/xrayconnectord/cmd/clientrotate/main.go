package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/realglebivanov/hstd/hstdlib"
	"github.com/realglebivanov/hstd/hstdlib/secret"
	"github.com/xtls/xray-core/infra/conf"
	"github.com/xtls/xray-core/proxy/vless"
)

func main() {
	if len(os.Args) != 2 {
		slog.Error("usage: clientrotate <secret>")
		os.Exit(1)
	}

	scrt, err := hstdlib.ParseHexSecret(os.Args[1])
	if err != nil {
		slog.Error("secret must be hex", "err", err)
		os.Exit(1)
	}
	uuids := secret.GenerateClientUUIDs(scrt)
	slog.Info("rotating client_id", "clients", len(uuids))

	if err := rotate(uuids); err != nil {
		slog.Error("rotate", "err", err)
		os.Exit(1)
	}
}

const configPath = "/usr/local/etc/xray/config.json"

func rotate(uuids []string) error {
	fi, err := os.Stat(configPath)
	if err != nil {
		return fmt.Errorf("stat config: %w", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("read config: %w", err)
	}

	var cfg conf.Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}

	if err := updateInbounds(&cfg, uuids); err != nil {
		return fmt.Errorf("update inbounds: %v", err)
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

func updateInbounds(cfg *conf.Config, uuids []string) error {
	for i, inbound := range cfg.InboundConfigs {
		if inbound.Protocol != "vless" {
			continue
		}
		if inbound.Settings == nil {
			continue
		}

		var settings conf.VLessInboundConfig
		if err := json.Unmarshal(*inbound.Settings, &settings); err != nil {
			return fmt.Errorf("parse vless settings: %w", err)
		}

		clients, err := buildClients(uuids)
		if err != nil {
			return fmt.Errorf("marshal client: %w", err)
		}
		settings.Clients = clients

		raw, err := json.Marshal(settings)
		if err != nil {
			return fmt.Errorf("marshal settings: %w", err)
		}
		rawMsg := json.RawMessage(raw)
		cfg.InboundConfigs[i].Settings = &rawMsg
	}

	return nil
}

func buildClients(uuids []string) ([]json.RawMessage, error) {
	var clients []json.RawMessage

	for _, uuid := range uuids {
		json, err := json.Marshal(&vless.Account{Id: uuid, Flow: "xtls-rprx-vision", Encryption: ""})
		if err != nil {
			return nil, fmt.Errorf("marshal client: %w", err)
		}
		clients = append(clients, json)
	}

	return clients, nil
}
