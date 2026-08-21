package totp

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"net/url"
	"strings"
	"time"
)

const (
	period  = 30
	digits  = 6
	window  = 1
	issuer  = "gra"
)

func RandomSecret() (string, error) {
	buf := make([]byte, 20)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(buf), nil
}

func URI(account, secret string) string {
	q := url.Values{}
	q.Set("secret", secret)
	q.Set("issuer", issuer)
	q.Set("algorithm", "SHA1")
	q.Set("digits", "6")
	q.Set("period", "30")
	return "otpauth://totp/" + url.PathEscape(issuer+":"+account) + "?" + q.Encode()
}

func Code(secret string, at time.Time) (string, error) {
	return hotp(secret, uint64(at.Unix()/period))
}

func Valid(secret, code string, at time.Time) bool {
	code = strings.TrimSpace(code)
	if len(code) != digits {
		return false
	}
	counter := uint64(at.Unix() / period)
	for i := -window; i <= window; i++ {
		n := int64(counter) + int64(i)
		if n < 0 {
			continue
		}
		got, err := hotp(secret, uint64(n))
		if err == nil && hmac.Equal([]byte(got), []byte(code)) {
			return true
		}
	}
	return false
}

func hotp(secret string, counter uint64) (string, error) {
	key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(strings.ToUpper(strings.TrimSpace(secret)))
	if err != nil {
		return "", err
	}
	var buf [8]byte
	binary.BigEndian.PutUint64(buf[:], counter)
	mac := hmac.New(sha1.New, key)
	_, _ = mac.Write(buf[:])
	sum := mac.Sum(nil)
	off := sum[len(sum)-1] & 0x0f
	bin := (int(sum[off])&0x7f)<<24 | int(sum[off+1])<<16 | int(sum[off+2])<<8 | int(sum[off+3])
	mod := 1
	for i := 0; i < digits; i++ {
		mod *= 10
	}
	return fmt.Sprintf("%0*d", digits, bin%mod), nil
}
