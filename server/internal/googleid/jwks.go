package googleid

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/big"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwksURL = "https://www.googleapis.com/oauth2/v3/certs"

type JWKSVerifier struct {
	Client   *http.Client
	Fallback Verifier

	mu      sync.Mutex
	keys    map[string]*rsa.PublicKey
	fetched time.Time
}

type googleClaims struct {
	Email         string `json:"email"`
	EmailVerified any    `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	jwt.RegisteredClaims
}

func NewVerifier(client *http.Client) Verifier {
	if client == nil {
		client = http.DefaultClient
	}
	return &JWKSVerifier{Client: client, Fallback: HTTPVerifier{Client: client}}
}

func (v *JWKSVerifier) Verify(ctx context.Context, rawToken, audience string) (Identity, error) {
	ident, err := v.verifyJWT(ctx, rawToken, audience)
	if err == nil {
		return ident, nil
	}
	if v != nil && v.Fallback != nil {
		return v.Fallback.Verify(ctx, rawToken, audience)
	}
	return Identity{}, err
}

func (v *JWKSVerifier) verifyJWT(ctx context.Context, rawToken, audience string) (Identity, error) {
	rawToken = strings.TrimSpace(rawToken)
	audience = strings.TrimSpace(audience)
	if rawToken == "" {
		return Identity{}, fmt.Errorf("missing id token")
	}
	if audience == "" {
		return Identity{}, fmt.Errorf("missing audience")
	}
	claims := &googleClaims{}
	tok, err := jwt.ParseWithClaims(rawToken, claims, func(t *jwt.Token) (any, error) {
		kid, _ := t.Header["kid"].(string)
		if kid == "" {
			return nil, fmt.Errorf("token missing kid")
		}
		return v.lookupKey(ctx, kid)
	}, jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}), jwt.WithAudience(audience), jwt.WithLeeway(30*time.Second))
	if err != nil || !tok.Valid {
		if err == nil {
			err = fmt.Errorf("invalid id token")
		}
		return Identity{}, err
	}
	iss := strings.TrimSpace(claims.Issuer)
	if iss != "https://accounts.google.com" && iss != "accounts.google.com" {
		return Identity{}, fmt.Errorf("token issuer mismatch")
	}
	if claims.Subject == "" {
		return Identity{}, fmt.Errorf("token missing subject")
	}
	return Identity{
		Subject:       claims.Subject,
		Email:         strings.ToLower(strings.TrimSpace(claims.Email)),
		EmailVerified: isTruthy(claims.EmailVerified),
		Name:          strings.TrimSpace(claims.Name),
		Picture:       strings.TrimSpace(claims.Picture),
	}, nil
}

func (v *JWKSVerifier) lookupKey(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	if k := v.cachedKey(kid); k != nil {
		return k, nil
	}
	if err := v.refresh(ctx); err != nil {
		return nil, err
	}
	if k := v.cachedKey(kid); k != nil {
		return k, nil
	}
	return nil, fmt.Errorf("unknown google signing key")
}

func (v *JWKSVerifier) cachedKey(kid string) *rsa.PublicKey {
	v.mu.Lock()
	defer v.mu.Unlock()
	if v.keys == nil || time.Since(v.fetched) > time.Hour {
		return nil
	}
	return v.keys[kid]
}

func (v *JWKSVerifier) refresh(ctx context.Context) error {
	client := v.Client
	if client == nil {
		client = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, jwksURL, nil)
	if err != nil {
		return err
	}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = res.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("google jwks status %d", res.StatusCode)
	}
	var doc struct {
		Keys []struct {
			Kid string `json:"kid"`
			N   string `json:"n"`
			E   string `json:"e"`
		} `json:"keys"`
	}
	if err := json.Unmarshal(body, &doc); err != nil {
		return fmt.Errorf("decode jwks: %w", err)
	}
	keys := make(map[string]*rsa.PublicKey, len(doc.Keys))
	for _, k := range doc.Keys {
		pub, err := rsaFromJWK(k.N, k.E)
		if err != nil || k.Kid == "" {
			continue
		}
		keys[k.Kid] = pub
	}
	if len(keys) == 0 {
		return fmt.Errorf("google jwks empty")
	}
	v.mu.Lock()
	v.keys = keys
	v.fetched = time.Now()
	v.mu.Unlock()
	return nil
}

func rsaFromJWK(n, e string) (*rsa.PublicKey, error) {
	nb, err := base64.RawURLEncoding.DecodeString(n)
	if err != nil {
		return nil, err
	}
	eb, err := base64.RawURLEncoding.DecodeString(e)
	if err != nil {
		return nil, err
	}
	if len(nb) == 0 || len(eb) == 0 {
		return nil, fmt.Errorf("invalid jwk")
	}
	exp := new(big.Int).SetBytes(eb)
	if !exp.IsInt64() {
		return nil, fmt.Errorf("invalid exponent")
	}
	expN := exp.Int64()
	if expN <= 0 || expN > math.MaxInt32 {
		return nil, fmt.Errorf("invalid exponent")
	}
	return &rsa.PublicKey{
		N: new(big.Int).SetBytes(nb),
		E: int(expN),
	}, nil
}
