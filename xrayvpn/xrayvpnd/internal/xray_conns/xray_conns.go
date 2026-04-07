package xray_conns

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/realglebivanov/hstd/hstdlib"
	"github.com/realglebivanov/hstd/hstdlib/httpclient"
	"github.com/realglebivanov/hstd/hstdlib/xrayconf"
	"github.com/realglebivanov/hstd/xrayvpnd/internal/config/repo"
)

func Add(db *repo.DB, rawURL string) error {
	cfg, err := decodeLink(rawURL)
	if err != nil {
		return err
	}
	return db.AddConn(cfg)
}

func SyncAll(db *repo.DB) (int, error) {
	subs, err := db.GetSubs()
	if err != nil {
		return 0, err
	}
	if len(subs) == 0 {
		return 0, fmt.Errorf("no subscriptions configured")
	}

	cfgs := make(map[string][]*xrayconf.Config)
	var total int
	var errs []error

	for _, sub := range subs {
		configs, err := fetchSub(sub.URL)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", sub.URL, err))
			continue
		}
		cfgs[sub.ID] = configs
		total += len(configs)
	}

	if total > 0 {
		if err := db.SyncConns(cfgs); err != nil {
			return 0, err
		}
	}

	if len(errs) > 0 {
		return total, fmt.Errorf("sync errors: %v", errs)
	}
	return total, nil
}

func fetchSub(url string) ([]*xrayconf.Config, error) {
	resp, err := httpclient.Direct.Get(url)
	if err != nil {
		resp, err = httpclient.Default.Get(url)
	}
	if err != nil {
		return nil, fmt.Errorf("fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	br := bufio.NewReader(resp.Body)
	first, err := br.Peek(1)
	if err != nil {
		return nil, fmt.Errorf("peek: %w", err)
	}

	if first[0] == '[' {
		return decodeJSON(br)
	}
	return decodeLinks(br)
}

func decodeJSON(r io.Reader) ([]*xrayconf.Config, error) {
	var configs []*xrayconf.Config
	if err := json.NewDecoder(r).Decode(&configs); err != nil {
		return nil, fmt.Errorf("decode JSON: %w", err)
	}
	return configs, nil
}

func decodeLinks(r io.Reader) ([]*xrayconf.Config, error) {
	scanner := bufio.NewScanner(base64.NewDecoder(base64.StdEncoding, r))
	var configs []*xrayconf.Config
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		cfg, err := decodeLink(line)
		if err != nil {
			return nil, fmt.Errorf("parse link %q: %w", line, err)
		}
		configs = append(configs, cfg)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan base64 links: %w", err)
	}
	return configs, nil
}

func decodeLink(rawURL string) (*xrayconf.Config, error) {
	l, err := xrayconf.ParseVLESSLink(rawURL)
	if err != nil {
		return nil, fmt.Errorf("parse link %q: %w", rawURL, err)
	}
	outbound := l.Outbound(hstdlib.ProxyTag)
	return &xrayconf.Config{
		Outbounds: []*xrayconf.Outbound{&outbound},
		Routing: &xrayconf.RoutingConfig{
			DomainStrategy: "IPIfNonMatch",
			Rules: []xrayconf.RouteRule{
				{Type: "field", OutboundTag: hstdlib.DirectTag, IP: []string{"geoip:ru", "geoip:private"}},
				{Type: "field", OutboundTag: hstdlib.DirectTag, Domain: []string{"geosite:category-ru"}},
				{Type: "field", OutboundTag: hstdlib.ProxyTag, Network: "tcp,udp"},
			},
		},
	}, nil
}
