package mailer

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	"go-react-shadcn/internal/models"
)

func UnsubToken(secret, kind string, userID uint) string {
	kind = models.NormalizeUserKind(kind)
	id := strconv.FormatUint(uint64(userID), 10)
	payload := kind + "." + id
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	return payload + "." + hex.EncodeToString(mac.Sum(nil)[:16])
}

func ParseUnsubToken(secret, token string) (string, uint, bool) {
	parts := strings.Split(strings.TrimSpace(token), ".")
	if len(parts) == 3 {
		kind, idStr, sig := parts[0], parts[1], parts[2]
		if kind != models.UserKindAdmin && kind != models.UserKindWeb {
			return "", 0, false
		}
		id, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil || id == 0 || sig == "" {
			return "", 0, false
		}
		want := UnsubToken(secret, kind, uint(id))
		if !hmac.Equal([]byte(want), []byte(kind+"."+idStr+"."+sig)) {
			return "", 0, false
		}
		return kind, uint(id), true
	}
	if len(parts) != 2 {
		return "", 0, false
	}
	idStr, sig := parts[0], parts[1]
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil || id == 0 || sig == "" {
		return "", 0, false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(idStr))
	want := idStr + "." + hex.EncodeToString(mac.Sum(nil)[:16])
	if !hmac.Equal([]byte(want), []byte(idStr+"."+sig)) {
		return "", 0, false
	}
	return models.UserKindWeb, uint(id), true
}

func UnsubLink(baseURL, secret, kind string, userID uint) string {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		base = "http://127.0.0.1:5173"
	}
	return base + "/unsubscribe?token=" + UnsubToken(secret, kind, userID)
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
