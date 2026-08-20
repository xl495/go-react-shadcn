package googleid

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var tokenInfoURL = "https://oauth2.googleapis.com/tokeninfo"

type Identity struct {
	Subject       string
	Email         string
	EmailVerified bool
	Name          string
	Picture       string
}

type Verifier interface {
	Verify(ctx context.Context, rawToken, audience string) (Identity, error)
}

type HTTPVerifier struct {
	Client *http.Client
}

type tokenInfo struct {
	Aud              string `json:"aud"`
	Sub              string `json:"sub"`
	Email            string `json:"email"`
	EmailVerified    any    `json:"email_verified"`
	Name             string `json:"name"`
	Picture          string `json:"picture"`
	Exp              any    `json:"exp"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

func (v HTTPVerifier) Verify(ctx context.Context, rawToken, audience string) (Identity, error) {
	rawToken = strings.TrimSpace(rawToken)
	audience = strings.TrimSpace(audience)
	if rawToken == "" {
		return Identity{}, fmt.Errorf("missing id token")
	}
	if audience == "" {
		return Identity{}, fmt.Errorf("missing audience")
	}
	client := v.Client
	if client == nil {
		client = http.DefaultClient
	}
	endpoint := tokenInfoURL + "?id_token=" + url.QueryEscape(rawToken)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return Identity{}, err
	}
	res, err := client.Do(req)
	if err != nil {
		return Identity{}, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return Identity{}, err
	}
	var info tokenInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return Identity{}, fmt.Errorf("decode tokeninfo: %w", err)
	}
	if info.Error != "" {
		msg := info.Error
		if info.ErrorDescription != "" {
			msg = info.ErrorDescription
		}
		return Identity{}, fmt.Errorf("google token: %s", msg)
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return Identity{}, fmt.Errorf("google tokeninfo status %d", res.StatusCode)
	}
	if info.Aud != audience {
		return Identity{}, fmt.Errorf("token audience mismatch")
	}
	if info.Sub == "" {
		return Identity{}, fmt.Errorf("token missing subject")
	}
	if exp := unixFromAny(info.Exp); exp > 0 && time.Now().Unix() >= exp {
		return Identity{}, fmt.Errorf("token expired")
	}
	email := strings.ToLower(strings.TrimSpace(info.Email))
	return Identity{
		Subject:       info.Sub,
		Email:         email,
		EmailVerified: isTruthy(info.EmailVerified),
		Name:          strings.TrimSpace(info.Name),
		Picture:       strings.TrimSpace(info.Picture),
	}, nil
}

func isTruthy(v any) bool {
	switch x := v.(type) {
	case bool:
		return x
	case string:
		s := strings.ToLower(strings.TrimSpace(x))
		return s == "true" || s == "1" || s == "yes"
	case float64:
		return x != 0
	default:
		return false
	}
}

func unixFromAny(v any) int64 {
	switch x := v.(type) {
	case string:
		n, err := strconv.ParseInt(strings.TrimSpace(x), 10, 64)
		if err != nil {
			return 0
		}
		return n
	case float64:
		return int64(x)
	case json.Number:
		n, err := x.Int64()
		if err != nil {
			return 0
		}
		return n
	default:
		return 0
	}
}
