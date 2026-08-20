package siteverify

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const (
	Recaptcha = "https://www.google.com/recaptcha/api/siteverify"
	Turnstile = "https://challenges.cloudflare.com/turnstile/v0/siteverify"
)

type Result struct {
	Success    bool     `json:"success"`
	Score      float64  `json:"score"`
	Action     string   `json:"action"`
	Hostname   string   `json:"hostname"`
	ErrorCodes []string `json:"error-codes"`
}

type Client struct {
	HTTP *http.Client
}

func (c *Client) Check(ctx context.Context, endpoint, secret, response, remoteIP string) (Result, error) {
	secret = strings.TrimSpace(secret)
	response = strings.TrimSpace(response)
	if secret == "" {
		return Result{}, fmt.Errorf("missing verify secret")
	}
	if response == "" {
		return Result{}, fmt.Errorf("missing captcha token")
	}
	form := url.Values{}
	form.Set("secret", secret)
	form.Set("response", response)
	if ip := strings.TrimSpace(remoteIP); ip != "" {
		form.Set("remoteip", ip)
	}
	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return Result{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := httpClient.Do(req)
	if err != nil {
		return Result{}, err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return Result{}, err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return Result{}, fmt.Errorf("siteverify status %d", res.StatusCode)
	}
	var out Result
	if err := json.Unmarshal(body, &out); err != nil {
		return Result{}, fmt.Errorf("decode siteverify: %w", err)
	}
	return out, nil
}
