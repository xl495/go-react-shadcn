package httpserver

import "testing"

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
