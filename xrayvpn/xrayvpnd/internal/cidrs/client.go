package cidrs

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

	"github.com/realglebivanov/hstd/xrayvpnd/internal/httpclient"
)

type source struct {
	Name string
	URL  string
}

var sources = []source{
	{"ripencc", "https://ftp.ripe.net/pub/stats/ripencc/delegated-ripencc-extended-latest"},
	{"apnic", "https://ftp.apnic.net/stats/apnic/delegated-apnic-extended-latest"},
	{"arin", "https://ftp.arin.net/pub/stats/arin/delegated-arin-extended-latest"},
	{"lacnic", "https://ftp.lacnic.net/pub/stats/lacnic/delegated-lacnic-extended-latest"},
	{"afrinic", "https://ftp.afrinic.net/pub/stats/afrinic/delegated-afrinic-extended-latest"},
}

func tryToFetchSource(src *source) ([]string, error) {
	cidrs, err := fetchSource(httpclient.Default, src)
	if err != nil {
		log.Printf("fetch source with default client %s: %v", src.Name, err)
		cidrs, err := fetchSource(httpclient.Direct, src)
		if err != nil {
			return nil, fmt.Errorf("fetch source with direct client %s: %v", src.Name, err)
		}
		return cidrs, nil
	}

	return cidrs, nil
}

func fetchSource(client *http.Client, src *source) ([]string, error) {
	log.Printf("fetching RU CIDRs from %s ...", src.Name)

	resp, err := client.Get(src.URL)
	if err != nil {
		return nil, fmt.Errorf("fetch %s: %w", src.URL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch %s: HTTP %d", src.URL, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", src.URL, err)
	}

	cidrs, err := parseCIDRs(body)
	if err != nil {
		return nil, err
	}

	log.Printf("fetched %d RU CIDRs from %s", len(cidrs), src.Name)
	return cidrs, nil
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
