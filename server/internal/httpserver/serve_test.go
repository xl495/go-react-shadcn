package httpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestListenAndServeHealthAndGracefulShutdown(t *testing.T) {
	app := testApp(t)
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	addr := ln.Addr().String()
	if err := ln.Close(); err != nil {
		t.Fatalf("close probe listener: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	errCh := make(chan error, 1)
	go func() {
		errCh <- app.ListenAndServe(ctx, addr)
	}()

	client := &http.Client{Timeout: 2 * time.Second}
	url := fmt.Sprintf("http://%s/health", addr)
	deadline := time.Now().Add(5 * time.Second)
	var lastErr error
	ready := false
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err != nil {
			lastErr = err
			time.Sleep(30 * time.Millisecond)
			continue
		}
		body, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			t.Fatalf("read health: %v", readErr)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("health status=%d body=%s", resp.StatusCode, body)
		}
		if !strings.Contains(string(body), "ok") {
			t.Fatalf("health body %q does not contain ok", body)
		}
		var env envelope
		if err := json.Unmarshal(body, &env); err != nil {
			t.Fatalf("health json: %v", err)
		}
		if env.Message != "ok" {
			t.Fatalf("message=%q", env.Message)
		}
		var data struct {
			Status string `json:"status"`
		}
		if err := json.Unmarshal(env.Data, &data); err != nil {
			t.Fatalf("health data: %v", err)
		}
		if data.Status != "ok" {
			t.Fatalf("status=%q", data.Status)
		}
		ready = true
		break
	}
	if !ready {
		t.Fatalf("health never became ready: %v", lastErr)
	}

	metrics, err := client.Get(fmt.Sprintf("http://%s/metrics", addr))
	if err != nil {
		t.Fatalf("metrics: %v", err)
	}
	metricsBody, _ := io.ReadAll(metrics.Body)
	_ = metrics.Body.Close()
	if metrics.StatusCode != http.StatusOK {
		t.Fatalf("metrics status=%d body=%s", metrics.StatusCode, metricsBody)
	}
	if !strings.Contains(string(metricsBody), "latch_http_requests_total") {
		t.Fatalf("metrics missing counter: %s", metricsBody)
	}

	cancel()
	select {
	case err := <-errCh:
		if err != nil {
			t.Fatalf("shutdown: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("server did not shut down")
	}

	if _, err := client.Get(url); err == nil {
		t.Fatal("expected connection error after shutdown")
	}
}
