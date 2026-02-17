package middleware

import (
	"bytes"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/itprodirect/go-hello-world/internal/metrics"
)

func okHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
}

func panicHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	})
}

func TestLogger(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(&buf, "", 0)
	handler := Logger(logger, okHandler())

	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	logLine := buf.String()
	if !strings.Contains(logLine, "GET") || !strings.Contains(logLine, "/hello") {
		t.Errorf("log line missing expected fields: %q", logLine)
	}
}

func TestRecover(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(&buf, "", 0)
	handler := Recover(logger, panicHandler())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
	if !strings.Contains(buf.String(), "PANIC") {
		t.Errorf("expected PANIC log, got: %q", buf.String())
	}
}

func TestRequestCounter(t *testing.T) {
	counters := metrics.NewCounters()
	handler := RequestCounter(counters, okHandler())

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/hello", nil))
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/hello", nil))

	if got := counters.Get("http_requests_total"); got != 2 {
		t.Errorf("http_requests_total = %d, want 2", got)
	}
	if got := counters.Get("path_hello_requests"); got != 2 {
		t.Errorf("path_hello_requests = %d, want 2", got)
	}
}

func TestAllowMethods(t *testing.T) {
	handler := AllowMethods([]string{http.MethodGet}, okHandler())

	getRec := httptest.NewRecorder()
	handler.ServeHTTP(getRec, httptest.NewRequest(http.MethodGet, "/", nil))
	if getRec.Code != http.StatusOK {
		t.Errorf("GET status = %d, want 200", getRec.Code)
	}

	postRec := httptest.NewRecorder()
	handler.ServeHTTP(postRec, httptest.NewRequest(http.MethodPost, "/", nil))
	if postRec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST status = %d, want 405", postRec.Code)
	}
}

func TestChain(t *testing.T) {
	counters := metrics.NewCounters()
	var buf bytes.Buffer
	logger := log.New(&buf, "", 0)

	handler := Chain(
		okHandler(),
		func(h http.Handler) http.Handler { return Logger(logger, h) },
		func(h http.Handler) http.Handler { return Recover(logger, h) },
		func(h http.Handler) http.Handler { return RequestCounter(counters, h) },
	)

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if counters.Get("http_requests_total") != 1 {
		t.Error("counter not incremented")
	}
	if !strings.Contains(buf.String(), "GET") {
		t.Error("logger not invoked")
	}
}
