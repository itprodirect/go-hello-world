package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunUnknownFlagReturnsTwo(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"--bogus"}, strings.NewReader(""), &stdout, &stderr)
	if code != 2 {
		t.Fatalf("code=%d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "flag provided but not defined") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestRunInvalidModeReturnsOne(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"--mode", "unknown"}, strings.NewReader("hello\n"), &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code=%d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "invalid mode/options") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestRunUpperFromStdin(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"--mode", "upper"}, strings.NewReader("hello\nworld\n"), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code=%d, want 0; stderr=%q", code, stderr.String())
	}
	if stdout.String() != "HELLO\nWORLD\n" {
		t.Fatalf("stdout=%q", stdout.String())
	}
}

func TestRunGrepRequiresMatch(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"--mode", "grep"}, strings.NewReader("hello\n"), &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code=%d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "--match required") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}

func TestRunJSONExtractAlias(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	input := "{\"name\":\"Nick\"}\n{\"name\":\"Go\"}\n"

	code := run([]string{"--mode", "json-extract", "--field", "name"}, strings.NewReader(input), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code=%d, want 0; stderr=%q", code, stderr.String())
	}
	if stdout.String() != "Nick\nGo\n" {
		t.Fatalf("stdout=%q", stdout.String())
	}
}

func TestRunConcurrentChainMode(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	input := "  a  \n  a  \n  b  \n"

	code := run([]string{"--mode", "chain", "--workers", "2"}, strings.NewReader(input), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code=%d, want 0; stderr=%q", code, stderr.String())
	}

	output := stdout.String()
	if !strings.Contains(output, "| a") || !strings.Contains(output, "| b") {
		t.Fatalf("unexpected chain output: %q", output)
	}
}

func TestRunInputOutputFiles(t *testing.T) {
	tempDir := t.TempDir()
	inPath := filepath.Join(tempDir, "in.txt")
	outPath := filepath.Join(tempDir, "out.txt")

	if err := os.WriteFile(inPath, []byte("HELLO\nWORLD\n"), 0o644); err != nil {
		t.Fatalf("write input: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := run([]string{"--mode", "lower", "--in", inPath, "--out", outPath}, strings.NewReader(""), &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code=%d, want 0; stderr=%q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "wrote 2 lines") {
		t.Fatalf("unexpected stderr summary: %q", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout should remain empty when --out is set, got %q", stdout.String())
	}

	data, err := os.ReadFile(outPath)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if string(data) != "hello\nworld\n" {
		t.Fatalf("output file content=%q", string(data))
	}
}

func TestBuildStagesFilterAlias(t *testing.T) {
	stages, err := buildStages("filter", "error", "", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(stages) != 1 {
		t.Fatalf("len(stages)=%d, want 1", len(stages))
	}
	if got := stages[0]("has error"); got == "" {
		t.Fatal("expected filter stage to keep matching line")
	}
	if got := stages[0]("all good"); got != "" {
		t.Fatal("expected filter stage to drop non-matching line")
	}
}

func TestBuildStagesReplaceRequiresOld(t *testing.T) {
	if _, err := buildStages("replace", "", "", "", "x"); err == nil {
		t.Fatal("expected error when --old is missing")
	}
}

func TestRunRejectsNegativeWorkers(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := run([]string{"--workers", "-1"}, strings.NewReader("hello\n"), &stdout, &stderr)
	if code != 1 {
		t.Fatalf("code=%d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "invalid workers") {
		t.Fatalf("unexpected stderr: %q", stderr.String())
	}
}
