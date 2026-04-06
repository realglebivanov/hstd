package sublink

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

type Sublink struct {
	Index int
	Hash  string
}

func New(idx int, rootSecret []byte) *Sublink {
	h := hmac.New(sha256.New, rootSecret)
	fmt.Fprintf(h, "subpath:%d", idx)
	return &Sublink{Index: idx, Hash: hex.EncodeToString(h.Sum(nil))}
}

func SubPath(idx int, rootSecret []byte) (string, error) {
	l := New(idx, rootSecret)
	j, err := json.Marshal(l)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(j), nil
}

func (sl *Sublink) IsValid(rootSecret []byte) bool {
	expected := New(sl.Index, rootSecret)
	return sl.Hash == expected.Hash
}

func Unmarshal(source string) (*Sublink, error) {
	jsn, err := hex.DecodeString(source)
	if err != nil {
		return nil, err
	}

	var l Sublink
	if err := json.Unmarshal(jsn, &l); err != nil {
		return nil, err
	}

	return &l, nil
}
