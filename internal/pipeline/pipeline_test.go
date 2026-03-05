package pipeline

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"
)

func TestRunSingleStage(t *testing.T) {
	input := "hello\nworld\n"
	r := strings.NewReader(input)
	var buf bytes.Buffer

	n, err := Run(context.Background(), r, &buf, strings.ToUpper)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 2 {
		t.Fatalf("wrote %d lines, want 2", n)
	}
	if buf.String() != "HELLO\nWORLD\n" {
		t.Fatalf("got %q", buf.String())
	}
}

func TestRunFilterStage(t *testing.T) {
	input := "keep this\ndrop this\nkeep that\n"
	r := strings.NewReader(input)
	var buf bytes.Buffer

	keepOnly := func(line string) string {
		if strings.HasPrefix(line, "keep") {
			return line
		}
		return ""
	}

	n, err := Run(context.Background(), r, &buf, keepOnly)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 2 {
		t.Fatalf("wrote %d lines, want 2", n)
	}
}

func TestRunCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	r := strings.NewReader("hello\n")
	var buf bytes.Buffer

	n, err := Run(ctx, r, &buf)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context.Canceled, got %v", err)
	}
	if n != 0 {
		t.Fatalf("wrote %d lines, want 0", n)
	}
}

func TestRunLongLineScannerBuffer(t *testing.T) {
	line := strings.Repeat("a", 70*1024)
	r := strings.NewReader(line + "\n")
	var buf bytes.Buffer

	n, err := Run(context.Background(), r, &buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 1 {
		t.Fatalf("wrote %d lines, want 1", n)
	}
	if buf.String() != line+"\n" {
		t.Fatalf("unexpected output length: got %d, want %d", len(buf.String()), len(line)+1)
	}
}

func TestChain(t *testing.T) {
	chained := Chain(strings.TrimSpace, strings.ToUpper)
	got := chained("  hello  ")
	if got != "HELLO" {
		t.Fatalf("got %q, want HELLO", got)
	}
}

func TestRunConcurrentBasic(t *testing.T) {
	input := strings.Repeat("hello\n", 100)
	r := strings.NewReader(input)
	var buf bytes.Buffer

	n, err := RunConcurrent(context.Background(), r, &buf, 4, strings.ToUpper)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 100 {
		t.Fatalf("wrote %d lines, want 100", n)
	}
}

func TestRunConcurrentWorkersBelowOne(t *testing.T) {
	input := "a\nb\n"
	r := strings.NewReader(input)
	var buf bytes.Buffer

	n, err := RunConcurrent(context.Background(), r, &buf, 0, strings.ToUpper)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 2 {
		t.Fatalf("wrote %d lines, want 2", n)
	}
}

func TestRunConcurrentContextTimeout(t *testing.T) {
	input := strings.Repeat("line\n", 200)
	r := strings.NewReader(input)
	var buf bytes.Buffer

	slowStage := func(line string) string {
		time.Sleep(10 * time.Millisecond)
		return line
	}

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()

	n, err := RunConcurrent(ctx, r, &buf, 2, slowStage)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected DeadlineExceeded, got %v", err)
	}
	if n >= 200 {
		t.Fatalf("expected partial output, got %d lines", n)
	}
}

func TestRunConcurrentWriteError(t *testing.T) {
	input := strings.Repeat("line\n", 50)
	r := strings.NewReader(input)
	w := &failAfterWriter{maxWrites: 1}

	n, err := RunConcurrent(context.Background(), r, w, 4)
	if err == nil {
		t.Fatal("expected write error")
	}
	if !strings.Contains(err.Error(), "write:") {
		t.Fatalf("unexpected error: %v", err)
	}
	if n < 1 {
		t.Fatalf("expected at least one successful write, got %d", n)
	}
}

type failAfterWriter struct {
	writes    int
	maxWrites int
}

func (w *failAfterWriter) Write(p []byte) (int, error) {
	if w.writes >= w.maxWrites {
		return 0, io.ErrClosedPipe
	}
	w.writes++
	return len(p), nil
}
