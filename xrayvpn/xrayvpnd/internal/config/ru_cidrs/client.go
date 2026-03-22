package ru_cidrs

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	mathbits "math/bits"
	"net"
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

	if len(errs) != 0 {
		return nil, fmt.Errorf("%d/%d RIRs failed", len(errs), len(cidrUrls))
	}

	log.Printf("fetched %d unique RU CIDRs", len(cidrs))
	return cidrs, nil
}

func fetchAllCidrs() <-chan *cidrResult {
	ch := make(chan *cidrResult, len(cidrUrls))
	client := &http.Client{Timeout: 30 * time.Second}
	for _, url := range cidrUrls {
		go fetchCidrs(client, url, ch)
	}
	return ch
}

func fetchCidrs(client *http.Client, url string, ch chan<- *cidrResult) {
	resp, err := client.Get(url)
	if err != nil {
		ch <- &cidrResult{err: fmt.Errorf("fetch %s: %w", url, err), url: url}
		return
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		ch <- &cidrResult{err: fmt.Errorf("fetch %s: %d", url, resp.StatusCode), url: url}
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		ch <- &cidrResult{err: fmt.Errorf("read %s: %w", url, err), url: url}
		return
	}

	cidrs, err := parseCIDRs(body)
	if err != nil {
		ch <- &cidrResult{err: err, url: url}
		return
	}

	ch <- &cidrResult{cidrs: cidrs, url: url}
}

func parseCIDRs(body []byte) ([]string, error) {
	var cidrs []string
	scanner := bufio.NewScanner(bytes.NewReader(body))

	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), "|")
		if len(fields) < 5 {
			continue
		}
		if fields[1] != "RU" || fields[2] != "ipv4" {
			continue
		}
		ip := net.ParseIP(fields[3]).To4()
		if ip == nil {
			continue
		}
		count, err := strconv.ParseUint(fields[4], 10, 32)
		if err != nil || count == 0 {
			continue
		}
		cidrs = append(cidrs, rangeToCIDRs(ip, uint(count))...)
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return cidrs, nil
}

func rangeToCIDRs(start net.IP, count uint) []string {
	blockStart := binary.BigEndian.Uint32(start)

	var cidrs []string
	for count > 0 {
		trailingZeros := mathbits.TrailingZeros32(blockStart)

		maxBits := min(trailingZeros, mathbits.Len(count)-1)
		blockSize := 1 << maxBits
		prefix := 32 - maxBits

		ip := make(net.IP, 4)
		binary.BigEndian.PutUint32(ip, blockStart)
		cidrs = append(cidrs, fmt.Sprintf("%s/%d", ip, prefix))

		blockStart += uint32(blockSize)
		count -= uint(blockSize)
	}
	return cidrs
}
