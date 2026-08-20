package mailer

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
)

func UnsubToken(secret string, userID uint) string {
	mac := hmac.New(sha256.New, []byte(secret))
	id := strconv.FormatUint(uint64(userID), 10)
	_, _ = mac.Write([]byte(id))
	return id + "." + hex.EncodeToString(mac.Sum(nil)[:16])
}

func ParseUnsubToken(secret, token string) (uint, bool) {
	idStr, sig, ok := strings.Cut(strings.TrimSpace(token), ".")
	if !ok || idStr == "" || sig == "" {
		return 0, false
	}
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id == 0 {
		return 0, false
	}
	want := UnsubToken(secret, uint(id))
	if !hmac.Equal([]byte(want), []byte(idStr+"."+sig)) {
		return 0, false
	}
	return uint(id), true
}

func UnsubLink(baseURL, secret string, userID uint) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		base = "http://127.0.0.1:5173"
	}
	return base + "/unsubscribe?token=" + UnsubToken(secret, userID)
}

func appendUnsubFooter(body, link string) string {
	if strings.Contains(body, "/unsubscribe?token=") {
		return body
	}
	if LooksHTML(body) {
		return strings.TrimRight(body, "\n") +
			`<p style="margin-top:24px;font-size:12px;color:#666">如不想再收到营销邮件，请<a href="` + link + `">点击退订</a>。</p>`
	}
	return fmt.Sprintf("%s\n\n——\n如不想再收到营销邮件，请点击退订：\n%s\n", strings.TrimRight(body, "\n"), link)
}
