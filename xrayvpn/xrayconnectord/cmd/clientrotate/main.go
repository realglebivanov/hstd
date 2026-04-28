package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/realglebivanov/hstd/hstdlib"
	"github.com/realglebivanov/hstd/hstdlib/dataloader/cidrs"
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

	uuids := secret.GenerateGracefulClientUUIDs(scrt)
	slog.Info("rotating client_id", "clients", len(uuids), "flow", flow)

	rules, err := loadRoutingRules()
	if err != nil {
		slog.Error("load routing rules, falling back to allow-all", "err", err)
		rules = []xrayconf.RouteRule{{
			Type:        "field",
			OutboundTag: hstdlib.DirectTag,
			Network:     "tcp,udp",
		}}
	}

	if err := rotate(uuids, flow, rules); err != nil {
		slog.Error("rotate", "err", err)
		os.Exit(1)
	}
}

const (
	xrayConfigPath         = "/usr/local/etc/xray/config.json"
	clientrotateConfigPath = "/etc/clientrotate/config.json"
)

type clientrotateConfig struct {
	RoutingRules []xrayconf.RouteRule `json:"routingRules"`
}

func loadRoutingRules() ([]xrayconf.RouteRule, error) {
	data, err := os.ReadFile(clientrotateConfigPath)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", clientrotateConfigPath, err)
	}
	var cfg clientrotateConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", clientrotateConfigPath, err)
	}

	ruCIDRs, errs := cidrs.FetchAll()
	if err := errors.Join(errs...); err != nil {
		return nil, fmt.Errorf("fetch RU CIDRs: %w", err)
	}
	slog.Info("fetched RU CIDRs", "count", len(ruCIDRs))

	inverted := xrayconf.InvertRules(cfg.RoutingRules)

	return xrayconf.ExpandRules(inverted, ruCIDRs), nil
}

func rotate(uuids []string, flow string, rules []xrayconf.RouteRule) error {
	fi, err := os.Stat(xrayConfigPath)
	if err != nil {
		return fmt.Errorf("stat config: %w", err)
	}

	data, err := os.ReadFile(xrayConfigPath)
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

	cfg.Outbounds = []*xrayconf.Outbound{
		{
			Tag:      hstdlib.BlockTag,
			Protocol: "blackhole",
		},
		{
			Tag:      hstdlib.DirectTag,
			Protocol: "freedom",
		},
	}
	cfg.Routing = &xrayconf.RoutingConfig{
		Rules: rules,
	}

	out, err := json.MarshalIndent(cfg, "", "    ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	out = append(out, '\n')

	if err := os.WriteFile(xrayConfigPath, out, fi.Mode()); err != nil {
		return fmt.Errorf("write config: %w", err)
	}

	slog.Info("updated", "path", xrayConfigPath)
	return nil
}
