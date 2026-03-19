package store

import (
	"fmt"
	"net/url"
)

func validateLink(rawLink string) error {
	u, err := url.Parse(rawLink)
	if err != nil {
		return fmt.Errorf("parse url: %w", err)
	}
	if u.Scheme != "vless" {
		return fmt.Errorf("unsupported scheme %q, only vless is supported", u.Scheme)
	}
	if u.Hostname() == "" {
		return fmt.Errorf("missing hostname")
	}
	if u.Port() == "" {
		return fmt.Errorf("missing port")
	}
	return nil
}
