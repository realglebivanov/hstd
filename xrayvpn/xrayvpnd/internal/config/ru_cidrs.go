package config

import (
	"bufio"
	"bytes"
	"fmt"
	"log"
	"strings"

	"github.com/realglebivanov/hstd/xrayvpnd/internal/config/ru_cidrs"
)

const ruCIDRsName = "ru_cidrs.txt"

func loadRuCIDRs() ([]string, error) {
	cr := readCache(ruCIDRsName)
	switch cr.State {
	case cacheFresh:
		return unmarshalCIDRs(cr.Data)
	case cacheStale:
		log.Print("will refresh RU CIDRs in background")
		go RefreshRuCIDRs()
		return unmarshalCIDRs(cr.Data)
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

func unmarshalCIDRs(data []byte) ([]string, error) {
	var cidrs []string

	scanner := bufio.NewScanner(bytes.NewReader(data))

	for scanner.Scan() {
		line := scanner.Text()

		if line != "" {
			cidrs = append(cidrs, line)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan CIDRs: %w", err)
	}

	log.Printf("loaded %d cached RU CIDRs", len(cidrs))
	return cidrs, nil
}
