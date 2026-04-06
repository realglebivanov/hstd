package store

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/realglebivanov/hstd/hstdlib/xrayconf"
)

type Sub struct {
	ID  string `json:"id"`
	URL string `json:"url"`
}

func NewSub(url string) *Sub {
	h := sha256.Sum256([]byte(url))
	return &Sub{ID: hex.EncodeToString(h[:4]), URL: url}
}

type Conn struct {
	ID     string          `json:"id"`
	Remark string          `json:"remark"`
	Config json.RawMessage `json:"config"`
	SubID  string          `json:"sub_id,omitempty"`
}

func NewConn(cfg *xrayconf.Config, subID string) (*Conn, error) {
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

	return &Conn{
		ID:     id,
		Remark: remark,
		Config: data,
		SubID:  subID,
	}, nil
}

func (l *Conn) Summary() string {
	return l.Remark
}
