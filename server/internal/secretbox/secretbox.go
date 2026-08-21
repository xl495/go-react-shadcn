package secretbox

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"log/slog"
	"strings"
)

const prefix = "enc:v1:"

func key(master string) []byte {
	sum := sha256.Sum256([]byte("latch-sys-config|" + master))
	return sum[:]
}

func Seal(master, plaintext string) (string, error) {
	if master == "" || plaintext == "" || strings.HasPrefix(plaintext, prefix) {
		return plaintext, nil
	}
	block, err := aes.NewCipher(key(master))
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	out := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return prefix + base64.RawStdEncoding.EncodeToString(out), nil
}

func Open(master, value string) (string, error) {
	if !strings.HasPrefix(value, prefix) {
		return value, nil
	}
	if master == "" {
		return "", errors.New("missing secret key")
	}
	raw, err := base64.RawStdEncoding.DecodeString(strings.TrimPrefix(value, prefix))
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key(master))
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	if len(raw) < gcm.NonceSize() {
		return "", errors.New("ciphertext too short")
	}
	nonce, ct := raw[:gcm.NonceSize()], raw[gcm.NonceSize():]
	plain, err := gcm.Open(nil, nonce, ct, nil)
	if err != nil {
		return "", err
	}
	return string(plain), nil
}

func IsSealed(value string) bool {
	return strings.HasPrefix(value, prefix)
}

func MustOpen(master, value string) string {
	out, err := Open(master, value)
	if err != nil {
		slog.Error("secretbox decrypt failed", "error", err)
		return ""
	}
	return out
}
