package link

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

type LinkInfo struct {
	Index   int
	Comment string
	Enabled bool
	Devices string
}

type Link struct {
	Index int
	Hash  string
}

func New(idx int, rootSecret []byte) *Link {
	return &Link{Index: idx, Hash: deriveHash(idx, rootSecret)}
}

func (l *Link) IsValid(rootSecret []byte) bool {
	return l.Hash == deriveHash(l.Index, rootSecret)
}

func (l *Link) Marshal() (string, error) {
	json, err := json.Marshal(l)
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(json), nil
}

func Unmarshal(source string) (*Link, error) {
	jsn, err := hex.DecodeString(source)
	if err != nil {
		return nil, err
	}

	var l Link
	if err := json.Unmarshal(jsn, &l); err != nil {
		return nil, err
	}

	return &l, nil
}

func deriveHash(index int, rootSecret []byte) string {
	secretMsg := fmt.Appendf(nil, "subpath:%d", index)
	secretMAC := hmac.New(sha256.New, rootSecret)
	secretMAC.Write(secretMsg)

	return hex.EncodeToString(secretMAC.Sum(nil))
}
