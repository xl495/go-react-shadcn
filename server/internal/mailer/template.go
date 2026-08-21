package mailer

import (
	"strings"

	"go-react-shadcn/internal/models"
)

func RenderMailTemplate(subject, body string, user *models.User, unsub string) (string, string) {
	nick, username, email := "", "", ""
	if user != nil {
		username = user.Username
		email = user.Email
		nick = user.Nickname
		if nick == "" {
			nick = user.Username
		}
	}
	r := strings.NewReplacer(
		"{{nickname}}", nick,
		"{{username}}", username,
		"{{email}}", email,
		"{{unsubscribe}}", unsub,
		"{{name}}", nick,
	)
	return r.Replace(subject), r.Replace(body)
}

func LooksHTML(s string) bool {
	t := strings.ToLower(s)
	return strings.Contains(t, "<html") ||
		strings.Contains(t, "<p") ||
		strings.Contains(t, "<div") ||
		strings.Contains(t, "<br") ||
		strings.Contains(t, "<h1") ||
		strings.Contains(t, "<h2") ||
		strings.Contains(t, "<table") ||
		strings.Contains(t, "<span") ||
		strings.Contains(t, "<a ")
}

func wrapHTML(body string) string {
	body = sanitizeHTML(body)
	if strings.Contains(strings.ToLower(body), "<html") {
		return body
	}
	return "<!DOCTYPE html><html><body style=\"font-family:sans-serif;line-height:1.55;color:#222\">" +
		body + "</body></html>"
}
