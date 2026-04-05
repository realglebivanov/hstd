package secret

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"time"

	"github.com/realglebivanov/hstd/hstdlib"
)

func GenerateGracefulClientUUIDs(rootSecret []byte) []string {
	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)

	uuids := make([]string, 0, hstdlib.XrayClientCount*2)
	for i := range hstdlib.XrayClientCount {
		uuids = append(uuids, generateClientUUID(i, rootSecret, now))
		uuids = append(uuids, generateClientUUID(i, rootSecret, yesterday))
	}

	return uuids
}

func GenerateClientUUID(i int, rootSecret []byte) string {
	return generateClientUUID(i, rootSecret, time.Now())
}

func generateClientUUID(i int, rootSecret []byte, t time.Time) string {
	secret := deriveSubscriptionSecret(i, rootSecret)

	epoch := t.UTC().Add(-3 * time.Hour)
	day := time.Date(epoch.Year(), epoch.Month(), epoch.Day(), 0, 0, 0, 0, time.UTC).Unix()

	h := sha256.New()
	binary.Write(h, binary.BigEndian, day)
	h.Write(secret)
	sum := h.Sum(nil)

	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		sum[0:4], sum[4:6], sum[6:8], sum[8:10], sum[10:16])
}

func deriveSubscriptionSecret(index int, rootSecret []byte) []byte {
	secretMsg := fmt.Appendf(nil, "secret:%d", index)
	secretMAC := hmac.New(sha256.New, rootSecret)
	secretMAC.Write(secretMsg)

	return secretMAC.Sum(nil)
}
