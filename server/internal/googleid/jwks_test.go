package googleid

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"math/big"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func TestJWKSVerifierAcceptsSignedToken(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	kid := "test-kid"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"keys": []map[string]string{{
				"kty": "RSA",
				"kid": kid,
				"n":   base64.RawURLEncoding.EncodeToString(priv.N.Bytes()),
				"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(priv.E)).Bytes()),
			}},
		})
	}))
	t.Cleanup(srv.Close)
	orig := jwksURL
	jwksURL = srv.URL
	t.Cleanup(func() { jwksURL = orig })

	claims := googleClaims{
		Email:         "Ada@Example.com",
		EmailVerified: true,
		Name:          "Ada",
		Picture:       "https://example.com/a.png",
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "sub-9",
			Audience:  jwt.ClaimStrings{"client-1"},
			Issuer:    "https://accounts.google.com",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = kid
	raw, err := tok.SignedString(priv)
	if err != nil {
		t.Fatal(err)
	}

	v := &JWKSVerifier{Client: srv.Client()}
	ident, err := v.Verify(context.Background(), raw, "client-1")
	if err != nil {
		t.Fatal(err)
	}
	if ident.Subject != "sub-9" || ident.Email != "ada@example.com" || !ident.EmailVerified || ident.Name != "Ada" {
		t.Fatalf("%+v", ident)
	}
}

func TestJWKSVerifierRejectsWrongAudience(t *testing.T) {
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	kid := "kid-2"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{
			"keys": []map[string]string{{
				"kid": kid,
				"n":   base64.RawURLEncoding.EncodeToString(priv.N.Bytes()),
				"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(priv.E)).Bytes()),
			}},
		})
	}))
	t.Cleanup(srv.Close)
	orig := jwksURL
	jwksURL = srv.URL
	t.Cleanup(func() { jwksURL = orig })

	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, googleClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   "sub-9",
			Audience:  jwt.ClaimStrings{"other"},
			Issuer:    "https://accounts.google.com",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	})
	tok.Header["kid"] = kid
	raw, err := tok.SignedString(priv)
	if err != nil {
		t.Fatal(err)
	}
	v := &JWKSVerifier{Client: srv.Client()}
	if _, err := v.Verify(context.Background(), raw, "client-1"); err == nil {
		t.Fatal("expected audience error")
	}
}
