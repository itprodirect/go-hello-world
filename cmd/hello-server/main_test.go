package main

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/itprodirect/go-hello-world/internal/config"
	"github.com/itprodirect/go-hello-world/internal/metrics"
)

func TestNewHandlerHelloFormal(t *testing.T) {
	cfg := config.DefaultConfig()
	counters := metrics.NewCounters()
	logger := log.New(&bytes.Buffer{}, "", 0)
	h := newHandler(cfg, logger, counters)

	req := httptest.NewRequest(http.MethodGet, "/hello?name=Nick&style=formal", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", rec.Code)
	}

	var resp helloResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if !strings.Contains(resp.Message, "Good day, Nick.") {
		t.Fatalf("message=%q, want formal greeting", resp.Message)
	}
	if resp.Count != 1 {
		t.Fatalf("count=%d, want 1", resp.Count)
	}
}

func TestNewHandlerHelloMethodNotAllowed(t *testing.T) {
	cfg := config.DefaultConfig()
	counters := metrics.NewCounters()
	logger := log.New(&bytes.Buffer{}, "", 0)
	h := newHandler(cfg, logger, counters)

	req := httptest.NewRequest(http.MethodPost, "/hello", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusMethodNotAllowed {
		t.Fatalf("status=%d, want 405", rec.Code)
	}
}

func TestNewHandlerHelloBadInput(t *testing.T) {
	cfg := config.DefaultConfig()
	counters := metrics.NewCounters()
	logger := log.New(&bytes.Buffer{}, "", 0)
	h := newHandler(cfg, logger, counters)

	req := httptest.NewRequest(http.MethodGet, "/hello?name=<script>", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status=%d, want 400", rec.Code)
	}
}

func TestNewHandlerMetricsEndpoint(t *testing.T) {
	cfg := config.DefaultConfig()
	counters := metrics.NewCounters()
	logger := log.New(&bytes.Buffer{}, "", 0)
	h := newHandler(cfg, logger, counters)

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/hello?name=Nick", nil))
	metricsRec := httptest.NewRecorder()
	h.ServeHTTP(metricsRec, httptest.NewRequest(http.MethodGet, "/metrics", nil))

	if metricsRec.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", metricsRec.Code)
	}
	body := metricsRec.Body.String()
	if !strings.Contains(body, "http_requests_total") {
		t.Fatalf("metrics missing http_requests_total: %q", body)
	}
	if !strings.Contains(body, "metrics_requests") {
		t.Fatalf("metrics missing metrics_requests: %q", body)
	}
}
