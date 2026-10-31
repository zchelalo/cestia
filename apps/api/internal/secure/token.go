package secure

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

func RandomToken(bytes int) (string, error) {
	buf := make([]byte, bytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate random token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func Hash(value string) []byte {
	sum := sha256.Sum256([]byte(value))
	return sum[:]
}

func Derive(secret, purpose, id string, generation int) string {
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte("cestia:" + purpose + ":v1:"))
	_, _ = mac.Write([]byte(id))
	_, _ = mac.Write([]byte{0})
	_, _ = mac.Write([]byte(fmt.Sprintf("%d", generation)))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
