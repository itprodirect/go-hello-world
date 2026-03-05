package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/itprodirect/go-hello-world/internal/checker"
)

func TestRunWithCheckerParseErrorReturnsTwo(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := runWithChecker([]string{"--nope"}, &stdout, &stderr, func(ctx context.Context, target checker.Target) checker.Result {
		return checker.Result{}
	})

	if code != 2 {
		t.Fatalf("code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "flag provided but not defined") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestRunWithCheckerSuccessReturnsZeroAndJSONLatencyMs(t *testing.T) {
	targetsPath := writeTargetsFile(t, []checker.Target{
		{Name: "a", URL: "https://example.com", Type: "http"},
		{Name: "b", Host: "localhost", Type: "dns"},
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := runWithChecker([]string{"--targets", targetsPath, "--json"}, &stdout, &stderr, func(ctx context.Context, target checker.Target) checker.Result {
		return checker.Result{
			Name:    target.Name,
			Type:    target.Type,
			Target:  target.URL + target.Host,
			Status:  "up",
			Latency: 2500 * time.Millisecond,
			Detail:  "ok",
		}
	})

	if code != 0 {
		t.Fatalf("code = %d, want 0; stderr=%q", code, stderr.String())
	}

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("json lines = %d, want 2; output=%q", len(lines), stdout.String())
	}

	for _, line := range lines {
		var got map[string]interface{}
		if err := json.Unmarshal([]byte(line), &got); err != nil {
			t.Fatalf("unmarshal %q: %v", line, err)
		}

		if got["status"] != "up" {
			t.Fatalf("status=%v, want up", got["status"])
		}
		latency, ok := got["latency_ms"].(float64)
		if !ok {
			t.Fatalf("latency_ms missing or non-number: %#v", got["latency_ms"])
		}
		if latency != 2500 {
			t.Fatalf("latency_ms=%.0f, want 2500", latency)
		}
	}

	if !strings.Contains(stderr.String(), "2 up | 0 down | 0 errors") {
		t.Fatalf("unexpected summary: %q", stderr.String())
	}
}

func TestRunWithCheckerFailureReturnsOne(t *testing.T) {
	targetsPath := writeTargetsFile(t, []checker.Target{
		{Name: "up-target", URL: "https://example.com", Type: "http"},
		{Name: "down-target", URL: "https://example.org", Type: "http"},
	})

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := runWithChecker([]string{"--targets", targetsPath}, &stdout, &stderr, func(ctx context.Context, target checker.Target) checker.Result {
		status := "up"
		if target.Name == "down-target" {
			status = "down"
		}
		return checker.Result{
			Name:    target.Name,
			Type:    target.Type,
			Target:  target.URL,
			Status:  status,
			Latency: 20 * time.Millisecond,
			Detail:  status,
		}
	})

	if code != 1 {
		t.Fatalf("code = %d, want 1; stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "1 up | 1 down | 0 errors") {
		t.Fatalf("unexpected summary: %q", stderr.String())
	}
}

func TestRunWithCheckerNoTargetsFileUsesDemoMessage(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := runWithChecker([]string{"--json"}, &stdout, &stderr, func(ctx context.Context, target checker.Target) checker.Result {
		return checker.Result{
			Name:    target.Name,
			Type:    target.Type,
			Target:  target.URL + target.Host,
			Status:  "up",
			Latency: 10 * time.Millisecond,
		}
	})

	if code != 0 {
		t.Fatalf("code = %d, want 0", code)
	}
	if !strings.Contains(stderr.String(), "No targets file provided") {
		t.Fatalf("expected demo-targets message, got: %q", stderr.String())
	}
}

func writeTargetsFile(t *testing.T, targets []checker.Target) string {
	t.Helper()

	data, err := json.Marshal(targets)
	if err != nil {
		t.Fatalf("marshal targets: %v", err)
	}

	path := filepath.Join(t.TempDir(), "targets.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write targets: %v", err)
	}

	return path
}
