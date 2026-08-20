package mailer

import (
	"testing"

	"go-react-shadcn/internal/models"
)

func TestRenderMailTemplate(t *testing.T) {
	user := &models.User{Nickname: "李访客", Username: "viewer", Email: "viewer@latch.local"}
	subject, body := RenderMailTemplate(
		"你好 {{nickname}}",
		"账号 {{username}} / {{email}}\n退订 {{unsubscribe}}",
		user,
		"http://example/unsub",
	)
	if subject != "你好 李访客" {
		t.Fatalf("subject=%q", subject)
	}
	if body != "账号 viewer / viewer@latch.local\n退订 http://example/unsub" {
		t.Fatalf("body=%q", body)
	}
}

func TestLooksHTML(t *testing.T) {
	if LooksHTML("plain hello") {
		t.Fatal("plain text should not look like html")
	}
	if !LooksHTML("<p>hello</p>") {
		t.Fatal("expected html")
	}
}
