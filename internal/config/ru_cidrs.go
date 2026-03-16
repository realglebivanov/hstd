package config

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/realglebivanov/xray-vpn/internal/config/ru_cidrs"
)

const cidrCachePath = cacheDir + "/ru_cidrs.txt"

func loadRuCIDRs() ([]string, error) {
	cr := readCache(cidrCachePath)
	switch cr.State {
	case cacheFresh:
		cidrs := unmarshalCIDRs(cr.Data)
		log.Printf("loaded %d cached RU CIDRs from %s", len(cidrs), cidrCachePath)
		return cidrs, nil
	case cacheStale:
		cidrs := unmarshalCIDRs(cr.Data)
		log.Printf("loaded %d stale RU CIDRs from %s, will refresh in background", len(cidrs), cidrCachePath)
		go RefreshRuCIDRs()
		return cidrs, nil
	case cacheMissing:
		return RefreshRuCIDRs()
	case cacheError:
		return nil, fmt.Errorf("read CIDR cache: %w", cr.Err)
	default:
		return nil, fmt.Errorf("unexpected cache state %d", cr.State)
	}
}

func RefreshRuCIDRs() ([]string, error) {
	cidrs, err := ru_cidrs.FetchRuCIDRs()
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(cidrCachePath, []byte(strings.Join(cidrs, "\n")+"\n"), 0700); err != nil {
		log.Printf("warning: failed to write CIDR cache: %v", err)
	} else {
		log.Printf("wrote %d CIDRs to %s", len(cidrs), cidrCachePath)
	}
	return cidrs, nil
}

func unmarshalCIDRs(data []byte) []string {
	var cidrs []string
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line != "" {
			cidrs = append(cidrs, line)
		}
	}
	return cidrs
}
