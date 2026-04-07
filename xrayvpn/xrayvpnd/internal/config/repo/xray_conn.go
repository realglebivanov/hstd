package repo

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/realglebivanov/hstd/hstdlib/xrayconf"
)

type XraySub struct {
	ID  string `db:"id"`
	URL string `db:"url"`
}

func NewXraySub(url string) *XraySub {
	h := sha256.Sum256([]byte(url))
	return &XraySub{ID: hex.EncodeToString(h[:4]), URL: url}
}

type XrayConnInfo struct {
	XrayConn
	Active bool `db:"active"`
}

type XrayConn struct {
	ID     string `db:"id"`
	Remark string `db:"remark"`
	Config string `db:"config"`
	SubID  string `db:"sub_id"`
}

func NewXrayConn(cfg *xrayconf.Config, subID string) (*XrayConn, error) {
	data, err := json.Marshal(cfg)
	if err != nil {
		return nil, fmt.Errorf("marshal config: %w", err)
	}

	if len(cfg.Outbounds) == 0 ||
		cfg.Outbounds[0].Settings == nil ||
		len(cfg.Outbounds[0].Settings.Vnext) == 0 {
		return nil, fmt.Errorf("no vnext destinations")
	}

	ob := cfg.Outbounds[0]
	vnext := &ob.Settings.Vnext[0]
	remark := fmt.Sprintf(
		"%s:%d (%s, %s)",
		vnext.Address,
		vnext.Port,
		ob.StreamSettings.Network,
		ob.StreamSettings.Security,
	)

	h := sha256.Sum256([]byte(remark))
	id := hex.EncodeToString(h[:4])

	return &XrayConn{
		ID:     id,
		Remark: remark,
		Config: string(data),
		SubID:  subID,
	}, nil
}

func (c *XrayConn) Summary() string {
	return c.Remark
}
