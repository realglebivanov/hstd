package config

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"strings"

	"github.com/realglebivanov/xray-vpn/internal/config/ru_cidrs"
)

const ruCIDRsName = "ru_cidrs.txt"

func loadRuCIDRs() ([]string, error) {
	cr := readCache(ruCIDRsName)
	switch cr.State {
	case cacheFresh:
		cidrs := unmarshalCIDRs(cr.Data)
		log.Printf("loaded %d cached RU CIDRs", len(cidrs))
		return cidrs, nil
	case cacheStale:
		cidrs := unmarshalCIDRs(cr.Data)
		log.Printf("loaded %d stale RU CIDRs, will refresh in background", len(cidrs))
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
	if err := writeCache(ruCIDRsName, []byte(strings.Join(cidrs, "\n")+"\n")); err != nil {
		log.Printf("warning: failed to write CIDR cache: %v", err)
	} else {
		log.Printf("wrote %d CIDRs to cache", len(cidrs))
	}
	return cidrs, nil
}

func unmarshalCIDRs(data []byte) []string {
	var cidrs []string

	scanner := bufio.NewScanner(bytes.NewReader(data))

	for scanner.Scan() {
		line := scanner.Text()

		if line != "" {
			cidrs = append(cidrs, line)
		}
	}
	return cidrs
}
