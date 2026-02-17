package checker

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestCheckHTTPHealthyServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	result := Check(context.Background(), Target{
		Name: "test-server",
		URL:  server.URL,
		Type: "http",
	})

	if result.Status != "up" {
		t.Errorf("Status = %q, want up (detail=%s)", result.Status, result.Detail)
	}
	if result.Latency <= 0 {
		t.Error("Latency should be > 0")
	}
}

func TestCheckHTTPDownServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	result := Check(context.Background(), Target{
		Name: "broken-server",
		URL:  server.URL,
		Type: "http",
	})

	if result.Status != "down" {
		t.Errorf("Status = %q, want down", result.Status)
	}
}

func TestCheckHTTPUnreachable(t *testing.T) {
	result := Check(context.Background(), Target{
		Name:    "unreachable",
		URL:     "http://192.0.2.1:1",
		Type:    "http",
		Timeout: 500,
	})

	if result.Status != "down" {
		t.Errorf("Status = %q, want down", result.Status)
	}
}

func TestCheckTCPOpenPort(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	host, port, err := net.SplitHostPort(server.Listener.Addr().String())
	if err != nil {
		t.Fatalf("SplitHostPort: %v", err)
	}

	portNum := server.Listener.Addr().(*net.TCPAddr).Port
	if port == "" || portNum <= 0 {
		t.Fatalf("invalid listener address: host=%q port=%q portNum=%d", host, port, portNum)
	}

	result := Check(context.Background(), Target{
		Name: "test-tcp",
		Host: host,
		Port: portNum,
		Type: "tcp",
	})

	if result.Status != "up" {
		t.Errorf("Status = %q, want up (detail=%s)", result.Status, result.Detail)
	}
}

func TestCheckDNSValidHost(t *testing.T) {
	result := Check(context.Background(), Target{
		Name: "dns-localhost",
		Host: "localhost",
		Type: "dns",
	})

	if result.Status != "up" {
		t.Errorf("Status = %q, want up (detail=%s)", result.Status, result.Detail)
	}
}

func TestCheckUnknownType(t *testing.T) {
	result := Check(context.Background(), Target{
		Name: "unknown",
		Type: "ftp",
	})

	if result.Status != "error" {
		t.Errorf("Status = %q, want error", result.Status)
	}
}

func TestCheckRespectsTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	start := time.Now()
	result := Check(context.Background(), Target{
		Name:    "slow-server",
		URL:     server.URL,
		Type:    "http",
		Timeout: 200,
	})
	elapsed := time.Since(start)

	if result.Status != "down" {
		t.Errorf("Status = %q, want down", result.Status)
	}
	if elapsed > time.Second {
		t.Errorf("elapsed = %s, expected timeout near 200ms", elapsed)
	}
}

func TestLoadTargets(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "targets.json")
	data := []byte(`[
		{"name":"site","url":"https://example.com","type":"http"},
		{"name":"dns","host":"localhost","type":"dns"}
	]`)

	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write targets: %v", err)
	}

	targets, err := LoadTargets(path)
	if err != nil {
		t.Fatalf("LoadTargets returned error: %v", err)
	}

	if len(targets) != 2 {
		t.Fatalf("len(targets) = %d, want 2", len(targets))
	}
	if targets[0].Name != "site" || targets[1].Type != "dns" {
		t.Fatalf("unexpected targets: %#v", targets)
	}
}

func TestStatusEmoji(t *testing.T) {
	tests := []struct {
		status string
		want   string
	}{
		{"up", "[OK]"},
		{"down", "[FAIL]"},
		{"error", "[ERR]"},
		{"unknown", "[ERR]"},
	}

	for _, tt := range tests {
		if got := StatusEmoji(tt.status); got != tt.want {
			t.Errorf("StatusEmoji(%q) = %q, want %q", tt.status, got, tt.want)
		}
	}
}
