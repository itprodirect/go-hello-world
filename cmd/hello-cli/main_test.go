package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestRunUnknownFlagReturnsTwo(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"--bogus"}, &stdout, &stderr)
	if code != 2 {
		t.Fatalf("code=%d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "flag provided but not defined") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestRunInvalidRepeatReturnsOne(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"--repeat", "0"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code=%d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "invalid input") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestRunJSONFormalStyleOutput(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"--name", "Nick", "--repeat", "2", "--style", "formal", "--json"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code=%d, want 0; stderr=%q", code, stderr.String())
	}

	lines := strings.Split(strings.TrimSpace(stdout.String()), "\n")
	if len(lines) != 2 {
		t.Fatalf("lines=%d, want 2; stdout=%q", len(lines), stdout.String())
	}

	for i, line := range lines {
		var item map[string]interface{}
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			t.Fatalf("unmarshal line %d: %v", i, err)
		}
		msg, _ := item["message"].(string)
		if !strings.Contains(msg, "Good day, Nick.") {
			t.Fatalf("message=%q, want formal greeting", msg)
		}
	}
}
