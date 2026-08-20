package i18n

import (
	"net/http"
	"testing"
)

func TestNormalize(t *testing.T) {
	cases := map[string]string{
		"":                  En,
		"en":                En,
		"en-US":             En,
		"zh":                ZhCN,
		"zh-CN":             ZhCN,
		"zh-Hans":           ZhCN,
		"zh-CN,zh;q=0.9,en": ZhCN,
		"en-US,en;q=0.8":    En,
		"fr-FR":             En,
	}
	for in, want := range cases {
		if got := Normalize(in); got != want {
			t.Fatalf("Normalize(%q)=%q want %q", in, got, want)
		}
	}
}

func TestFromRequestPrefersXLocale(t *testing.T) {
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept-Language", "en")
	req.Header.Set("X-Locale", "zh-CN")
	if got := FromRequest(req); got != ZhCN {
		t.Fatalf("got %q", got)
	}
}

func TestErrorZhAndFallback(t *testing.T) {
	if got := Error(ZhCN, 40103, "invalid credentials"); got != "用户名或密码错误" {
		t.Fatalf("zh 40103=%q", got)
	}
	if got := Error(En, 40103, "invalid credentials"); got != "invalid credentials" {
		t.Fatalf("en 40103=%q", got)
	}
	if got := Error(ZhCN, 99999, "custom fallback"); got != "custom fallback" {
		t.Fatalf("unknown code=%q", got)
	}
}
