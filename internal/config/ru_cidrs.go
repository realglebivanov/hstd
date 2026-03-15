package config

import (
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

var cidrUrls = []string{
	"https://ftp.ripe.net/pub/stats/ripencc/delegated-ripencc-extended-latest",
	"https://ftp.apnic.net/stats/apnic/delegated-apnic-extended-latest",
	"https://ftp.arin.net/pub/stats/arin/delegated-arin-extended-latest",
	"https://ftp.lacnic.net/pub/stats/lacnic/delegated-lacnic-extended-latest",
	"https://ftp.afrinic.net/pub/stats/afrinic/delegated-afrinic-extended-latest",
}

func fetchRuCIDRs() ([]string, error) {
	log.Println("fetching Russian CIDRs from all RIRs...")

	type result struct {
		cidrs []string
		err   error
		url   string
	}

	results := make(chan result, len(cidrUrls))
	client := &http.Client{Timeout: 30 * time.Second}

	for _, url := range cidrUrls {
		go func(url string) {
			cidrs, err := fetchAndParseRIR(client, url)
			results <- result{cidrs, err, url}
		}(url)
	}

	seen := make(map[string]struct{})
	var cidrs []string

	for range cidrUrls {
		r := <-results
		if r.err != nil {
			log.Printf("warning: failed to fetch %s: %v", r.url, r.err)
			continue
		}
		for _, cidr := range r.cidrs {
			if _, ok := seen[cidr]; !ok {
				seen[cidr] = struct{}{}
				cidrs = append(cidrs, cidr)
			}
		}
	}

	log.Printf("fetched %d unique RU CIDRs", len(cidrs))
	return cidrs, nil
}

func fetchAndParseRIR(client *http.Client, url string) ([]string, error) {
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", url, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", url, err)
	}

	var cidrs []string
	for line := range strings.SplitSeq(string(body), "\n") {
		fields := strings.Split(line, "|")
		if len(fields) < 5 {
			continue
		}
		if fields[1] != "RU" || fields[2] != "ipv4" {
			continue
		}
		ip := fields[3]
		count, err := strconv.Atoi(fields[4])
		if err != nil || count == 0 {
			continue
		}
		bits := 32 - int(math.Log2(float64(count)))
		cidrs = append(cidrs, fmt.Sprintf("%s/%d", ip, bits))
	}

	return cidrs, nil
}
