package client

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"log/slog"
	mathbits "math/bits"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/realglebivanov/hstd/hstdlib/httpclient"
)

type Source struct {
	Name string
	URL  string
}

var Sources = []Source{
	{"ripencc", "https://ftp.ripe.net/pub/stats/ripencc/delegated-ripencc-extended-latest"},
	{"apnic", "https://ftp.apnic.net/stats/apnic/delegated-apnic-extended-latest"},
	{"arin", "https://ftp.arin.net/pub/stats/arin/delegated-arin-extended-latest"},
	{"lacnic", "https://ftp.lacnic.net/pub/stats/lacnic/delegated-lacnic-extended-latest"},
	{"afrinic", "https://ftp.afrinic.net/pub/stats/afrinic/delegated-afrinic-extended-latest"},
}

func FetchSources(srcs []Source, fn func(*Source) *FetchResult) ([]string, []error) {
	results := make([]*FetchResult, len(srcs))
	var wg sync.WaitGroup
	for i, src := range srcs {
		wg.Go(func() { results[i] = fn(&src) })
	}
	wg.Wait()
	return toCIDRs(results)
}

func FetchSource(src *Source) *FetchResult {
	slog.Info("fetching RU CIDRs", "src", src.Name)

	resp, err := httpclient.Default.Get(src.URL)
	if err != nil {
		slog.Warn("failed to fetch", "src", src.Name, "err", err)
		return &FetchResult{Err: err}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		err := fmt.Errorf("fetch %s: HTTP %d", src.URL, resp.StatusCode)
		slog.Warn("failed to fetch", "src", src.Name, "err", err)
		return &FetchResult{Err: err}
	}

	cidrs, err := parseCIDRs(resp.Body)
	if err != nil {
		slog.Warn("failed to parse", "src", src.Name, "err", err)
		return &FetchResult{Err: err}
	}

	slog.Info("fetched RU CIDRs", "count", len(cidrs), "src", src.Name)
	return &FetchResult{CIDRs: cidrs}
}

func parseCIDRs(r io.Reader) ([]string, error) {
	var cidrs []string
	scanner := bufio.NewScanner(r)

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
