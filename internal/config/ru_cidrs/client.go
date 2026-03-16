package ru_cidrs

import (
	"fmt"
	"io"
	"log"
	mathbits "math/bits"
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

type cidrResult struct {
	cidrs []string
	err   error
	url   string
}

func FetchRuCIDRs() ([]string, error) {
	log.Println("fetching Russian CIDRs from all RIRs...")

	results := fetchAllCidrs()

	seen := make(map[string]struct{})
	var cidrs []string
	var errs []error

	for range cidrUrls {
		r := <-results
		if r.err != nil {
			log.Printf("warning: failed to fetch %s: %v", r.url, r.err)
			errs = append(errs, r.err)
			continue
		}
		for _, cidr := range r.cidrs {
			if _, ok := seen[cidr]; !ok {
				seen[cidr] = struct{}{}
				cidrs = append(cidrs, cidr)
			}
		}
	}

	log.Printf("fetched %d unique RU CIDRs (%d/%d RIRs failed)", len(cidrs), len(errs), len(cidrUrls))
	return cidrs, nil
}

func fetchAllCidrs() <-chan cidrResult {
	ch := make(chan cidrResult, len(cidrUrls))
	client := &http.Client{Timeout: 30 * time.Second}
	for _, url := range cidrUrls {
		go fetchCidrs(client, url, ch)
	}
	return ch
}

func fetchCidrs(client *http.Client, url string, ch chan<- cidrResult) {
	resp, err := client.Get(url)
	if err != nil {
		ch <- cidrResult{err: fmt.Errorf("fetch %s: %w", url, err), url: url}
		return
	}
	if resp.StatusCode != http.StatusOK {
		ch <- cidrResult{err: fmt.Errorf("fetch %s: %d", url, resp.StatusCode), url: url}
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		ch <- cidrResult{err: fmt.Errorf("read %s: %w", url, err), url: url}
		return
	}

	ch <- cidrResult{cidrs: parseCIDRs(body), url: url}
}

func parseCIDRs(body []byte) []string {
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
		bits := 32 - mathbits.Len(uint(count-1))
		cidrs = append(cidrs, fmt.Sprintf("%s/%d", ip, bits))
	}
	return cidrs
}
