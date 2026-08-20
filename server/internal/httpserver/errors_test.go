package httpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestErrorCodeRegistry(t *testing.T) {
	if CodePasswordTooShort != 40016 {
		t.Fatalf("CodePasswordTooShort=%d want 40016", CodePasswordTooShort)
	}
	if CodeInvalidDictValue != 40015 {
		t.Fatalf("CodeInvalidDictValue=%d want 40015", CodeInvalidDictValue)
	}
	if CodeAccountLocked != 40310 {
		t.Fatalf("CodeAccountLocked=%d want 40310", CodeAccountLocked)
	}
	seen := map[int]string{}
	for name, code := range map[string]int{
		"CodePasswordTooShort": CodePasswordTooShort,
		"CodeInvalidDictValue": CodeInvalidDictValue,
		"CodeAccountLocked":    CodeAccountLocked,
		"CodeBadCredentials":   CodeBadCredentials,
		"CodeMissingToken":     CodeMissingToken,
		"CodeInvalidToken":     CodeInvalidToken,
	} {
		if prev, ok := seen[code]; ok {
			t.Fatalf("duplicate code %d for %s and %s", code, prev, name)
		}
		seen[code] = name
	}
}

func TestFailLocalizesMessage(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/x", func(c *gin.Context) {
		fail(c, http.StatusUnauthorized, CodeBadCredentials, "invalid credentials")
	})

	zh := httptest.NewRecorder()
	reqZH, _ := http.NewRequest(http.MethodGet, "/x", nil)
	reqZH.Header.Set("Accept-Language", "zh-CN")
	r.ServeHTTP(zh, reqZH)
	if zh.Code != http.StatusUnauthorized {
		t.Fatalf("status=%d", zh.Code)
	}
	var envZH body
	if err := json.NewDecoder(zh.Body).Decode(&envZH); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if envZH.Message != "用户名或密码错误" {
		t.Fatalf("zh message=%q", envZH.Message)
	}

	en := httptest.NewRecorder()
	reqEN, _ := http.NewRequest(http.MethodGet, "/x", nil)
	reqEN.Header.Set("Accept-Language", "en")
	r.ServeHTTP(en, reqEN)
	var envEN body
	if err := json.NewDecoder(en.Body).Decode(&envEN); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if envEN.Message != "invalid credentials" {
		t.Fatalf("en message=%q", envEN.Message)
	}
}
