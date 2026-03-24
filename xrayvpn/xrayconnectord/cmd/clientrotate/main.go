package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/realglebivanov/hstd/hstdlib"
	"github.com/xtls/xray-core/infra/conf"
	"github.com/xtls/xray-core/proxy/vless"
)

func main() {
	if len(os.Args) != 2 {
		log.Fatal("usage: clientrotate <secret>")
	}

	secret, err := strconv.ParseUint(os.Args[1], 10, 64)
	if err != nil {
		log.Fatalf("secret must be an integer: %v", err)
	}
	uuid := hstdlib.GenerateClientUUID(secret)
	log.Printf("rotating client_id to %s", uuid)

	if err := rotate(uuid); err != nil {
		log.Fatalf("rotate: %v", err)
	}
}

const configPath = "/usr/local/etc/xray/config.json"

func rotate(uuid string) error {
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

	if err := updateInbounds(&cfg, uuid); err != nil {
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

	log.Printf("updated %s", configPath)
	return nil
}

func updateInbounds(cfg *conf.Config, uuid string) error {
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

		client, err := json.Marshal(&vless.Account{Id: uuid, Flow: "xtls-rprx-vision", Encryption: ""})
		if err != nil {
			return fmt.Errorf("marshal client: %w", err)
		}
		settings.Clients = []json.RawMessage{client}

		raw, err := json.Marshal(settings)
		if err != nil {
			return fmt.Errorf("marshal settings: %w", err)
		}
		rawMsg := json.RawMessage(raw)
		cfg.InboundConfigs[i].Settings = &rawMsg
	}

	return nil
}
