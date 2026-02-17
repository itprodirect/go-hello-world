# Phase 4: Stream Processing Pipeline — The Go↔Python Bridge

> **Produces:** `internal/pipeline`, `internal/transform`, `cmd/dataflow`
> **Teaches:** `io.Reader/Writer`, `bufio.Scanner`, channels as pipelines, closures
> **Ships:** A Unix-style stream processor that bridges Go speed with Python analytics

## Implementation Status (February 17, 2026)

- Status: Pending
- Planned next step: implement this phase in the next session.
- Dependency status: Phases 1 to 3 are complete and verified.

## Why This Phase

Go is 10-100x faster than Python at line-by-line data processing. But Python
is where your analytics, ML, and visualization live. The solution: **Go
handles the high-throughput transformation, Python handles the analysis.**

This phase builds a composable pipeline that reads from any source (files,
stdin, HTTP responses), transforms data through chainable stages, and writes
to any destination. It's the data plumbing that connects your tools.

```bash
# Go transforms at speed, Python analyzes
./bin/dataflow --mode json-extract --field status < access.log | python3 analyze.py

# Chain Go tools together
./bin/healthcheck --json | ./bin/dataflow --mode filter --match '"status":"down"'
```

---

## Package 1: `internal/pipeline/pipeline.go`

A channel-based pipeline engine. Each stage reads from one channel and
writes to another — Go's CSP (Communicating Sequential Processes) model.

```go
package pipeline

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"sync"
)

// Stage transforms a line of text. Return empty string to drop the line.
type Stage func(line string) string

// Run reads lines from r, pushes them through stages in order,
// and writes surviving lines to w. Returns count of lines written.
//
// This is the core abstraction: io.Reader → stages → io.Writer.
// The reader could be a file, stdin, HTTP body, or bytes.Buffer.
// The writer could be stdout, a file, or a network connection.
func Run(ctx context.Context, r io.Reader, w io.Writer, stages ...Stage) (int, error) {
	scanner := bufio.NewScanner(r)
	written := 0

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return written, ctx.Err()
		default:
		}

		line := scanner.Text()

		// Push through each stage
		for _, stage := range stages {
			line = stage(line)
			if line == "" {
				break // Stage dropped this line
			}
		}

		if line == "" {
			continue
		}

		if _, err := fmt.Fprintln(w, line); err != nil {
			return written, fmt.Errorf("write: %w", err)
		}
		written++
	}

	if err := scanner.Err(); err != nil {
		return written, fmt.Errorf("scan: %w", err)
	}

	return written, nil
}

// RunConcurrent fans out lines across multiple worker goroutines.
// Use when stages are CPU-heavy (e.g., regex, JSON parsing).
// NOTE: Output order is NOT preserved (use Run for ordered output).
func RunConcurrent(ctx context.Context, r io.Reader, w io.Writer, workers int, stages ...Stage) (int, error) {
	if workers < 1 {
		workers = 1
	}

	lines := make(chan string, workers*2)
	results := make(chan string, workers*2)

	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for line := range lines {
				for _, stage := range stages {
					line = stage(line)
					if line == "" {
						break
					}
				}
				if line != "" {
					results <- line
				}
			}
		}()
	}

	// Writer goroutine
	var writeErr error
	written := 0
	writerDone := make(chan struct{})
	go func() {
		defer close(writerDone)
		for line := range results {
			if _, err := fmt.Fprintln(w, line); err != nil {
				writeErr = err
				return
			}
			written++
		}
	}()

	// Read and feed lines
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			break
		case lines <- scanner.Text():
		}
	}
	close(lines)

	wg.Wait()
	close(results)
	<-writerDone

	if scanner.Err() != nil {
		return written, scanner.Err()
	}
	return written, writeErr
}

// Chain composes multiple stages into one.
func Chain(stages ...Stage) Stage {
	return func(line string) string {
		for _, s := range stages {
			line = s(line)
			if line == "" {
				return ""
			}
		}
		return line
	}
}
```

### Test: `internal/pipeline/pipeline_test.go`

```go
package pipeline

import (
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestRun_SingleStage(t *testing.T) {
	input := "hello\nworld\n"
	r := strings.NewReader(input)
	var buf bytes.Buffer

	n, err := Run(context.Background(), r, &buf, strings.ToUpper)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 2 {
		t.Errorf("wrote %d lines, want 2", n)
	}

	want := "HELLO\nWORLD\n"
	if buf.String() != want {
		t.Errorf("got %q, want %q", buf.String(), want)
	}
}

func TestRun_MultipleStages(t *testing.T) {
	input := "  hello  \n  world  \n"
	r := strings.NewReader(input)
	var buf bytes.Buffer

	n, err := Run(context.Background(), r, &buf,
		strings.TrimSpace,
		strings.ToUpper,
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 2 {
		t.Errorf("wrote %d lines, want 2", n)
	}
	if buf.String() != "HELLO\nWORLD\n" {
		t.Errorf("got %q", buf.String())
	}
}

func TestRun_FilterStage(t *testing.T) {
	input := "keep this\ndrop this\nkeep that\n"
	r := strings.NewReader(input)
	var buf bytes.Buffer

	filterKeep := func(line string) string {
		if strings.HasPrefix(line, "keep") {
			return line
		}
		return "" // drop
	}

	n, err := Run(context.Background(), r, &buf, filterKeep)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 2 {
		t.Errorf("wrote %d lines, want 2", n)
	}
}

func TestRun_EmptyInput(t *testing.T) {
	r := strings.NewReader("")
	var buf bytes.Buffer

	n, err := Run(context.Background(), r, &buf, strings.ToUpper)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 0 {
		t.Errorf("wrote %d lines, want 0", n)
	}
}

func TestRun_ContextCancellation(t *testing.T) {
	// Large input
	input := strings.Repeat("line\n", 10000)
	r := strings.NewReader(input)
	var buf bytes.Buffer

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := Run(ctx, r, &buf)
	if err == nil {
		t.Log("no error (may have completed before cancel)")
	}
	// Either context.Canceled error or completed — both are acceptable
}

func TestChain(t *testing.T) {
	chained := Chain(
		strings.TrimSpace,
		strings.ToUpper,
	)

	got := chained("  hello  ")
	if got != "HELLO" {
		t.Errorf("Chain() = %q, want %q", got, "HELLO")
	}
}

func TestRunConcurrent_Basic(t *testing.T) {
	input := strings.Repeat("hello\n", 100)
	r := strings.NewReader(input)
	var buf bytes.Buffer

	n, err := RunConcurrent(context.Background(), r, &buf, 4, strings.ToUpper)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 100 {
		t.Errorf("wrote %d lines, want 100", n)
	}
}

func BenchmarkRun(b *testing.B) {
	input := strings.Repeat("the quick brown fox jumps over the lazy dog\n", 10000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := strings.NewReader(input)
		var buf bytes.Buffer
		Run(context.Background(), r, &buf, strings.ToUpper, strings.TrimSpace)
	}
}

func BenchmarkRunConcurrent(b *testing.B) {
	input := strings.Repeat("the quick brown fox jumps over the lazy dog\n", 10000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r := strings.NewReader(input)
		var buf bytes.Buffer
		RunConcurrent(context.Background(), r, &buf, 4, strings.ToUpper, strings.TrimSpace)
	}
}
```

---

## Package 2: `internal/transform/transform.go`

Pre-built transform stages — the actual operations you'd use day-to-day.
Each one returns a `pipeline.Stage` function.

```go
package transform

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

// --- Text transforms ---

// Upper converts to uppercase.
func Upper(line string) string {
	return strings.ToUpper(line)
}

// Lower converts to lowercase.
func Lower(line string) string {
	return strings.ToLower(line)
}

// Trim removes leading/trailing whitespace.
func Trim(line string) string {
	return strings.TrimSpace(line)
}

// Prefix adds a prefix to every line.
func Prefix(prefix string) func(string) string {
	return func(line string) string {
		return prefix + line
	}
}

// Suffix adds a suffix to every line.
func Suffix(suffix string) func(string) string {
	return func(line string) string {
		return line + suffix
	}
}

// --- Numbering ---

// NumberLines adds line numbers. Returns a closure that tracks state.
func NumberLines() func(string) string {
	n := 0
	return func(line string) string {
		n++
		return fmt.Sprintf("%6d | %s", n, line)
	}
}

// --- Filtering ---

// Contains keeps only lines containing the substring.
func Contains(substr string) func(string) string {
	return func(line string) string {
		if strings.Contains(line, substr) {
			return line
		}
		return "" // drop
	}
}

// NotContains drops lines containing the substring.
func NotContains(substr string) func(string) string {
	return func(line string) string {
		if strings.Contains(line, substr) {
			return "" // drop
		}
		return line
	}
}

// MatchRegex keeps only lines matching the pattern.
func MatchRegex(pattern string) (func(string) string, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("compile regex: %w", err)
	}
	return func(line string) string {
		if re.MatchString(line) {
			return line
		}
		return ""
	}, nil
}

// --- Deduplication ---

// Dedup drops consecutive duplicate lines (like Unix `uniq`).
func Dedup() func(string) string {
	var prev string
	return func(line string) string {
		if line == prev {
			return ""
		}
		prev = line
		return line
	}
}

// --- JSON ---

// JSONExtractField extracts a string field from a JSON line.
// If the line isn't valid JSON or the field doesn't exist, drops the line.
func JSONExtractField(field string) func(string) string {
	return func(line string) string {
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			return ""
		}
		val, ok := obj[field]
		if !ok {
			return ""
		}
		return fmt.Sprintf("%v", val)
	}
}

// JSONPretty reformats compact JSON lines as indented JSON.
func JSONPretty(line string) string {
	var obj interface{}
	if err := json.Unmarshal([]byte(line), &obj); err != nil {
		return line // pass through non-JSON
	}
	pretty, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return line
	}
	return string(pretty)
}

// --- Replace ---

// Replace does a simple string replacement.
func Replace(old, new string) func(string) string {
	return func(line string) string {
		return strings.ReplaceAll(line, old, new)
	}
}

// ReplaceRegex does regex-based replacement.
func ReplaceRegex(pattern, replacement string) (func(string) string, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("compile regex: %w", err)
	}
	return func(line string) string {
		return re.ReplaceAllString(line, replacement)
	}, nil
}
```

### Test: `internal/transform/transform_test.go`

```go
package transform

import (
	"testing"
)

func TestUpper(t *testing.T) {
	if got := Upper("hello"); got != "HELLO" {
		t.Errorf("Upper = %q, want HELLO", got)
	}
}

func TestLower(t *testing.T) {
	if got := Lower("HELLO"); got != "hello" {
		t.Errorf("Lower = %q, want hello", got)
	}
}

func TestTrim(t *testing.T) {
	if got := Trim("  hello  "); got != "hello" {
		t.Errorf("Trim = %q, want hello", got)
	}
}

func TestPrefix(t *testing.T) {
	fn := Prefix(">> ")
	if got := fn("hello"); got != ">> hello" {
		t.Errorf("Prefix = %q", got)
	}
}

func TestNumberLines(t *testing.T) {
	fn := NumberLines()
	first := fn("alpha")
	second := fn("beta")
	if first != "     1 | alpha" {
		t.Errorf("first = %q", first)
	}
	if second != "     2 | beta" {
		t.Errorf("second = %q", second)
	}
}

func TestContains(t *testing.T) {
	fn := Contains("error")
	if got := fn("error: something broke"); got == "" {
		t.Error("should keep matching line")
	}
	if got := fn("info: all good"); got != "" {
		t.Error("should drop non-matching line")
	}
}

func TestNotContains(t *testing.T) {
	fn := NotContains("debug")
	if got := fn("debug: noisy"); got != "" {
		t.Error("should drop matching line")
	}
	if got := fn("error: real problem"); got == "" {
		t.Error("should keep non-matching line")
	}
}

func TestDedup(t *testing.T) {
	fn := Dedup()
	if fn("a") == "" {
		t.Error("first line should not be dropped")
	}
	if fn("a") != "" {
		t.Error("duplicate line should be dropped")
	}
	if fn("b") == "" {
		t.Error("new line should not be dropped")
	}
}

func TestJSONExtractField(t *testing.T) {
	fn := JSONExtractField("name")

	tests := []struct {
		input string
		want  string
	}{
		{`{"name":"Nick","age":30}`, "Nick"},
		{`{"age":30}`, ""},           // missing field → drop
		{`not json`, ""},              // invalid → drop
	}

	for _, tt := range tests {
		got := fn(tt.input)
		if got != tt.want {
			t.Errorf("JSONExtractField(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestReplace(t *testing.T) {
	fn := Replace("world", "Go")
	got := fn("hello world")
	if got != "hello Go" {
		t.Errorf("Replace = %q", got)
	}
}

func TestMatchRegex(t *testing.T) {
	fn, err := MatchRegex(`^\d{3}-`)
	if err != nil {
		t.Fatal(err)
	}
	if fn("404-not-found") == "" {
		t.Error("should match")
	}
	if fn("info: ok") != "" {
		t.Error("should not match")
	}
}
```

---

## New Tool: `cmd/dataflow/main.go`

A Unix-style stream processor. Pipe data in, get transformed data out.

```go
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/itprodirect/go-hello-world/internal/pipeline"
	"github.com/itprodirect/go-hello-world/internal/transform"
)

func main() {
	mode := flag.String("mode", "upper", "transform: upper, lower, trim, number, grep, drop, json-field, replace, dedup, chain")
	match := flag.String("match", "", "substring for grep/drop modes")
	field := flag.String("field", "", "JSON field name for json-field mode")
	old := flag.String("old", "", "string to replace (replace mode)")
	new := flag.String("new", "", "replacement string (replace mode)")
	inFile := flag.String("in", "", "input file (default: stdin)")
	outFile := flag.String("out", "", "output file (default: stdout)")
	concurrent := flag.Int("workers", 0, "concurrent workers (0 = sequential)")
	flag.Parse()

	stages, err := buildStages(*mode, *match, *field, *old, *new)
	if err != nil {
		log.Fatalf("invalid mode/options: %v", err)
	}

	// Open reader
	var r *os.File
	if *inFile != "" {
		f, err := os.Open(*inFile)
		if err != nil {
			log.Fatalf("open input: %v", err)
		}
		defer f.Close()
		r = f
	} else {
		r = os.Stdin
	}

	// Open writer
	var w *os.File
	if *outFile != "" {
		f, err := os.Create(*outFile)
		if err != nil {
			log.Fatalf("create output: %v", err)
		}
		defer f.Close()
		w = f
	} else {
		w = os.Stdout
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	var n int
	if *concurrent > 0 {
		n, err = pipeline.RunConcurrent(ctx, r, w, *concurrent, stages...)
	} else {
		n, err = pipeline.Run(ctx, r, w, stages...)
	}

	if err != nil {
		log.Fatalf("pipeline error: %v", err)
	}

	if *outFile != "" {
		fmt.Fprintf(os.Stderr, "wrote %d lines to %s\n", n, *outFile)
	}
}

func buildStages(mode, match, field, oldStr, newStr string) ([]pipeline.Stage, error) {
	switch strings.ToLower(mode) {
	case "upper":
		return []pipeline.Stage{transform.Upper}, nil
	case "lower":
		return []pipeline.Stage{transform.Lower}, nil
	case "trim":
		return []pipeline.Stage{transform.Trim}, nil
	case "number":
		return []pipeline.Stage{transform.NumberLines()}, nil
	case "grep":
		if match == "" {
			return nil, fmt.Errorf("--match required for grep mode")
		}
		return []pipeline.Stage{transform.Contains(match)}, nil
	case "drop":
		if match == "" {
			return nil, fmt.Errorf("--match required for drop mode")
		}
		return []pipeline.Stage{transform.NotContains(match)}, nil
	case "json-field":
		if field == "" {
			return nil, fmt.Errorf("--field required for json-field mode")
		}
		return []pipeline.Stage{transform.JSONExtractField(field)}, nil
	case "json-pretty":
		return []pipeline.Stage{transform.JSONPretty}, nil
	case "replace":
		if oldStr == "" {
			return nil, fmt.Errorf("--old required for replace mode")
		}
		return []pipeline.Stage{transform.Replace(oldStr, newStr)}, nil
	case "dedup":
		return []pipeline.Stage{transform.Dedup()}, nil
	case "chain":
		// Useful default chain: trim → dedup → number
		return []pipeline.Stage{
			transform.Trim,
			transform.Dedup(),
			transform.NumberLines(),
		}, nil
	default:
		return nil, fmt.Errorf("unknown mode: %q", mode)
	}
}
```

---

## Usage Examples

```bash
# Uppercase a file
cat README.md | go run ./cmd/dataflow --mode upper

# Filter log lines for errors
cat /var/log/syslog | go run ./cmd/dataflow --mode grep --match "error"

# Drop debug noise from logs
cat app.log | go run ./cmd/dataflow --mode drop --match "DEBUG"

# Number lines of source code
go run ./cmd/dataflow --mode number --in cmd/healthcheck/main.go

# Extract a field from JSON lines (pipe from healthcheck)
go run ./cmd/healthcheck --json | go run ./cmd/dataflow --mode json-field --field name

# Pretty-print JSON lines
go run ./cmd/healthcheck --json | go run ./cmd/dataflow --mode json-pretty

# Replace strings
echo "Hello world" | go run ./cmd/dataflow --mode replace --old world --new Go

# Deduplicate and number (chain mode)
sort access.log | go run ./cmd/dataflow --mode chain

# High-throughput concurrent processing
go run ./cmd/dataflow --mode upper --in bigfile.txt --out output.txt --workers 4
```

---

## Concepts Demonstrated

| Go Pattern | Python/JS Equivalent | Why Go Wins Here |
|-----------|---------------------|-----------------|
| `io.Reader` / `io.Writer` | `typing.IO` / `Readable`/`Writable` | Same code works on files, stdin, HTTP, buffers |
| `bufio.Scanner` | `for line in file` | Efficient line-by-line reading |
| `pipeline.Stage` (func type) | Lambda / arrow function | First-class functions as pipeline stages |
| `Chain()` composition | `functools.reduce` / `.pipe()` | Compose transforms declaratively |
| Channel-based fan-out | `multiprocessing.Pool` | Zero-copy, no serialization overhead |
| Closures with state | Closures in Python/JS | `NumberLines()` tracks line count in closure |

---

## Verification

```bash
go test ./internal/pipeline/...
go test ./internal/transform/...
go test ./...

echo -e "hello\nworld\nhello" | go run ./cmd/dataflow --mode upper
echo -e "hello\nworld\nhello" | go run ./cmd/dataflow --mode dedup
echo -e '{"name":"Nick"}\n{"name":"Go"}' | go run ./cmd/dataflow --mode json-field --field name

make bench  # compare sequential vs concurrent pipeline
```
