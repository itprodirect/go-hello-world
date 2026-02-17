# Phase 3: Concurrent Health Checker — Go's Killer Feature

> **Produces:** `internal/checker`, `internal/workerpool`, `cmd/healthcheck`
> **Teaches:** Goroutines, channels, fan-out/fan-in, `context.WithTimeout`, `sync.WaitGroup`
> **Ships:** A real endpoint monitoring tool

## Implementation Status (February 17, 2026)

- Status: Complete
- Implemented:
  - `internal/workerpool/workerpool.go`
  - `internal/workerpool/workerpool_test.go`
  - `internal/checker/checker.go`
  - `internal/checker/checker_test.go`
  - `cmd/healthcheck/main.go`
  - `targets.example.json`
- Notes:
  - The implementation resolves spec TODOs (for example, `LoadTargets` uses `os.ReadFile` + `json.Unmarshal`).
  - CLI added input validation for worker/timeout minimum values.
- Verification: included in full-repo `go test ./...`, `go vet ./...`, and `go build ./...` pass.

## Why This Is the Phase That Matters Most

This is **the** reason to use Go instead of Python for tools like this.

Python's `asyncio` or `ThreadPoolExecutor` can check 50 URLs concurrently.
Go can check 5,000 with goroutines that cost ~2KB each. No event loop, no
`await`, no callback hell. Just `go func()` and channels.

This phase builds a real health checker you can use to monitor IT Pro Direct
sites, verify API endpoints before deployment, or scan bulk URLs for the
Link Safety Hub.

---

## Package 1: `internal/workerpool/workerpool.go`

A **generic, reusable** fan-out/fan-in worker pool. This is the most
valuable lego in the whole repo — it can power any concurrent batch job.

```go
package workerpool

import (
	"context"
	"sync"
)

// TaskFunc processes a single input item and returns a result.
// This is the function your workers will run.
type TaskFunc[In any, Out any] func(ctx context.Context, input In) Out

// Pool manages a fixed number of concurrent workers.
type Pool[In any, Out any] struct {
	workers int
}

// New creates a pool with the given number of workers.
func New[In any, Out any](workers int) *Pool[In, Out] {
	if workers < 1 {
		workers = 1
	}
	return &Pool[In, Out]{workers: workers}
}

// Run fans out inputs across workers and collects all results.
// Order is NOT guaranteed — use the index in your result if you need ordering.
// The returned slice has one result per input, but in arbitrary order.
func (p *Pool[In, Out]) Run(ctx context.Context, inputs []In, fn TaskFunc[In, Out]) []Out {
	if len(inputs) == 0 {
		return nil
	}

	jobs := make(chan In)
	results := make(chan Out, len(inputs))

	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < p.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for input := range jobs {
				select {
				case <-ctx.Done():
					return
				default:
					results <- fn(ctx, input)
				}
			}
		}()
	}

	// Feed jobs
	go func() {
		for _, input := range inputs {
			select {
			case <-ctx.Done():
				break
			case jobs <- input:
			}
		}
		close(jobs)
	}()

	// Wait for all workers to finish, then close results
	go func() {
		wg.Wait()
		close(results)
	}()

	// Collect
	var out []Out
	for result := range results {
		out = append(out, result)
	}

	return out
}
```

### Test: `internal/workerpool/workerpool_test.go`

```go
package workerpool

import (
	"context"
	"fmt"
	"sort"
	"sync/atomic"
	"testing"
	"time"
)

func TestPoolRun_Basic(t *testing.T) {
	pool := New[int, string](4)
	inputs := []int{1, 2, 3, 4, 5}

	results := pool.Run(context.Background(), inputs, func(ctx context.Context, n int) string {
		return fmt.Sprintf("item_%d", n)
	})

	if len(results) != len(inputs) {
		t.Fatalf("got %d results, want %d", len(results), len(inputs))
	}

	// Sort for deterministic comparison
	sort.Strings(results)
	want := []string{"item_1", "item_2", "item_3", "item_4", "item_5"}
	for i, r := range results {
		if r != want[i] {
			t.Errorf("results[%d] = %q, want %q", i, r, want[i])
		}
	}
}

func TestPoolRun_EmptyInput(t *testing.T) {
	pool := New[int, int](4)
	results := pool.Run(context.Background(), nil, func(ctx context.Context, n int) int {
		return n
	})
	if results != nil {
		t.Fatalf("expected nil for empty input, got %v", results)
	}
}

func TestPoolRun_Concurrency(t *testing.T) {
	// Verify that work actually runs concurrently.
	pool := New[int, int](4)
	inputs := make([]int, 20)
	for i := range inputs {
		inputs[i] = i
	}

	var maxConcurrent atomic.Int32
	var current atomic.Int32

	results := pool.Run(context.Background(), inputs, func(ctx context.Context, n int) int {
		c := current.Add(1)
		// Track max concurrent workers seen
		for {
			old := maxConcurrent.Load()
			if c <= old || maxConcurrent.CompareAndSwap(old, c) {
				break
			}
		}
		time.Sleep(10 * time.Millisecond) // Simulate work
		current.Add(-1)
		return n * 2
	})

	if len(results) != 20 {
		t.Fatalf("got %d results, want 20", len(results))
	}

	if maxConcurrent.Load() < 2 {
		t.Error("expected concurrent execution, but max concurrent was < 2")
	}
}

func TestPoolRun_ContextCancellation(t *testing.T) {
	pool := New[int, int](2)
	inputs := make([]int, 100)
	for i := range inputs {
		inputs[i] = i
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	results := pool.Run(ctx, inputs, func(ctx context.Context, n int) int {
		time.Sleep(20 * time.Millisecond)
		return n
	})

	// With 2 workers and 20ms per task, we can't finish 100 items in 50ms.
	// We should get significantly fewer results.
	if len(results) >= 100 {
		t.Errorf("expected early termination, got all %d results", len(results))
	}
}

func BenchmarkPoolRun(b *testing.B) {
	pool := New[int, int](8)
	inputs := make([]int, 1000)
	for i := range inputs {
		inputs[i] = i
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pool.Run(context.Background(), inputs, func(ctx context.Context, n int) int {
			return n * 2
		})
	}
}
```

---

## Package 2: `internal/checker/checker.go`

Health check logic for HTTP endpoints, TCP ports, and DNS resolution.
Each check type is a function that fits the `workerpool.TaskFunc` signature.

```go
package checker

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"time"
)

// Target defines what to check.
type Target struct {
	Name     string `json:"name"`
	URL      string `json:"url,omitempty"`       // for HTTP checks
	Host     string `json:"host,omitempty"`      // for TCP/DNS checks
	Port     int    `json:"port,omitempty"`      // for TCP checks
	Type     string `json:"type"`                // "http", "tcp", "dns"
	Timeout  int    `json:"timeout_ms,omitempty"` // per-target timeout in ms
}

// Result is the outcome of a single health check.
type Result struct {
	Name     string        `json:"name"`
	Type     string        `json:"type"`
	Target   string        `json:"target"`
	Status   string        `json:"status"`  // "up", "down", "error"
	Latency  time.Duration `json:"latency_ms"`
	Detail   string        `json:"detail,omitempty"`
	TLS      *TLSInfo      `json:"tls,omitempty"`
}

// TLSInfo captures certificate details for HTTPS endpoints.
type TLSInfo struct {
	Subject   string    `json:"subject"`
	Issuer    string    `json:"issuer"`
	NotAfter  time.Time `json:"not_after"`
	DaysLeft  int       `json:"days_left"`
}

// Check runs the appropriate check based on Target.Type.
// This is the TaskFunc you pass to workerpool.Pool.Run().
func Check(ctx context.Context, target Target) Result {
	timeout := time.Duration(target.Timeout) * time.Millisecond
	if timeout <= 0 {
		timeout = 5 * time.Second
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	switch target.Type {
	case "http":
		return checkHTTP(ctx, target)
	case "tcp":
		return checkTCP(ctx, target)
	case "dns":
		return checkDNS(ctx, target)
	default:
		return Result{
			Name:   target.Name,
			Type:   target.Type,
			Target: target.URL,
			Status: "error",
			Detail: fmt.Sprintf("unknown check type: %q", target.Type),
		}
	}
}

func checkHTTP(ctx context.Context, target Target) Result {
	start := time.Now()
	result := Result{
		Name:   target.Name,
		Type:   "http",
		Target: target.URL,
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, target.URL, nil)
	if err != nil {
		result.Status = "error"
		result.Detail = fmt.Sprintf("build request: %v", err)
		result.Latency = time.Since(start)
		return result
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
		// Don't follow redirects — just check the immediate response
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	resp, err := client.Do(req)
	result.Latency = time.Since(start)

	if err != nil {
		result.Status = "down"
		result.Detail = err.Error()
		return result
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		result.Status = "up"
	} else {
		result.Status = "down"
	}
	result.Detail = fmt.Sprintf("HTTP %d", resp.StatusCode)

	// Capture TLS info if available
	if resp.TLS != nil && len(resp.TLS.PeerCertificates) > 0 {
		cert := resp.TLS.PeerCertificates[0]
		daysLeft := int(time.Until(cert.NotAfter).Hours() / 24)
		result.TLS = &TLSInfo{
			Subject:  cert.Subject.CommonName,
			Issuer:   cert.Issuer.CommonName,
			NotAfter: cert.NotAfter,
			DaysLeft: daysLeft,
		}
	}

	return result
}

func checkTCP(ctx context.Context, target Target) Result {
	start := time.Now()
	addr := fmt.Sprintf("%s:%d", target.Host, target.Port)
	result := Result{
		Name:   target.Name,
		Type:   "tcp",
		Target: addr,
	}

	var d net.Dialer
	conn, err := d.DialContext(ctx, "tcp", addr)
	result.Latency = time.Since(start)

	if err != nil {
		result.Status = "down"
		result.Detail = err.Error()
		return result
	}
	conn.Close()

	result.Status = "up"
	result.Detail = "connection successful"

	// If this is an HTTPS port, try to grab the TLS cert
	if target.Port == 443 || target.Port == 8443 {
		tlsConn, err := tls.DialWithDialer(
			&net.Dialer{Timeout: 3 * time.Second},
			"tcp", addr, &tls.Config{InsecureSkipVerify: true},
		)
		if err == nil {
			defer tlsConn.Close()
			state := tlsConn.ConnectionState()
			if len(state.PeerCertificates) > 0 {
				cert := state.PeerCertificates[0]
				daysLeft := int(time.Until(cert.NotAfter).Hours() / 24)
				result.TLS = &TLSInfo{
					Subject:  cert.Subject.CommonName,
					Issuer:   cert.Issuer.CommonName,
					NotAfter: cert.NotAfter,
					DaysLeft: daysLeft,
				}
			}
		}
	}

	return result
}

func checkDNS(ctx context.Context, target Target) Result {
	start := time.Now()
	result := Result{
		Name:   target.Name,
		Type:   "dns",
		Target: target.Host,
	}

	resolver := &net.Resolver{}
	addrs, err := resolver.LookupHost(ctx, target.Host)
	result.Latency = time.Since(start)

	if err != nil {
		result.Status = "down"
		result.Detail = err.Error()
		return result
	}

	result.Status = "up"
	result.Detail = fmt.Sprintf("resolved to %v", addrs)
	return result
}

// --- Config helpers ---

// LoadTargets reads a JSON targets file.
func LoadTargets(path string) ([]Target, error) {
	data, err := json.ReadFile(path)
	// Note: json doesn't have ReadFile; use os.ReadFile + json.Unmarshal
	return nil, err
}

// NOTE TO CODEX: Replace the LoadTargets function above with this correct version:
//
// func LoadTargets(path string) ([]Target, error) {
//     data, err := os.ReadFile(path)
//     if err != nil {
//         return nil, fmt.Errorf("read targets file: %w", err)
//     }
//     var targets []Target
//     if err := json.Unmarshal(data, &targets); err != nil {
//         return nil, fmt.Errorf("parse targets file: %w", err)
//     }
//     return targets, nil
// }

// StatusEmoji returns a visual indicator for terminal output.
func StatusEmoji(status string) string {
	switch status {
	case "up":
		return "[OK]"
	case "down":
		return "[FAIL]"
	default:
		return "[ERR]"
	}
}
```

### Test: `internal/checker/checker_test.go`

```go
package checker

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCheckHTTP_HealthyServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))
	defer server.Close()

	target := Target{
		Name: "test-server",
		URL:  server.URL,
		Type: "http",
	}

	result := Check(context.Background(), target)

	if result.Status != "up" {
		t.Errorf("Status = %q, want %q (detail: %s)", result.Status, "up", result.Detail)
	}
	if result.Latency <= 0 {
		t.Error("Latency should be > 0")
	}
}

func TestCheckHTTP_DownServer(t *testing.T) {
	// Use a server that returns 500
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	target := Target{
		Name: "broken-server",
		URL:  server.URL,
		Type: "http",
	}

	result := Check(context.Background(), target)

	if result.Status != "down" {
		t.Errorf("Status = %q, want %q", result.Status, "down")
	}
}

func TestCheckHTTP_Unreachable(t *testing.T) {
	target := Target{
		Name:    "unreachable",
		URL:     "http://192.0.2.1:1", // RFC 5737 TEST-NET — guaranteed unreachable
		Type:    "http",
		Timeout: 500, // 500ms timeout
	}

	result := Check(context.Background(), target)

	if result.Status != "down" {
		t.Errorf("Status = %q, want %q", result.Status, "down")
	}
}

func TestCheckTCP_OpenPort(t *testing.T) {
	// httptest.NewServer opens a TCP port we can check
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer server.Close()

	// Extract host and port from server URL
	// server.Listener.Addr() gives us "127.0.0.1:PORT"
	addr := server.Listener.Addr().String()
	host, _, _ := splitHostPort(addr)

	target := Target{
		Name: "test-tcp",
		Host: host,
		Port: server.Listener.Addr().(*net.TCPAddr).Port,
		Type: "tcp",
	}

	result := Check(context.Background(), target)

	if result.Status != "up" {
		t.Errorf("Status = %q, want %q (detail: %s)", result.Status, "up", result.Detail)
	}
}

// NOTE TO CODEX: The test above uses splitHostPort and net.TCPAddr. Add the
// necessary imports:  "net" and the following helper:
//
// func splitHostPort(addr string) (string, string, error) {
//     return net.SplitHostPort(addr)
// }
//
// Or simplify the test by directly using server.Listener.Addr().(*net.TCPAddr)
// to get .IP.String() and .Port separately.

func TestCheckDNS_ValidHost(t *testing.T) {
	target := Target{
		Name: "dns-localhost",
		Host: "localhost",
		Type: "dns",
	}

	result := Check(context.Background(), target)

	if result.Status != "up" {
		t.Errorf("Status = %q, want %q (detail: %s)", result.Status, "up", result.Detail)
	}
}

func TestCheckUnknownType(t *testing.T) {
	target := Target{
		Name: "unknown",
		Type: "ftp",
	}
	result := Check(context.Background(), target)
	if result.Status != "error" {
		t.Errorf("Status = %q, want %q", result.Status, "error")
	}
}

func TestCheck_RespectsTimeout(t *testing.T) {
	// Slow server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	target := Target{
		Name:    "slow-server",
		URL:     server.URL,
		Type:    "http",
		Timeout: 200, // 200ms — should timeout before the 2s response
	}

	start := time.Now()
	result := Check(context.Background(), target)
	elapsed := time.Since(start)

	if result.Status != "down" {
		t.Errorf("Status = %q, want %q (should timeout)", result.Status, "down")
	}
	if elapsed > 1*time.Second {
		t.Errorf("took %s, should have timed out around 200ms", elapsed)
	}
}
```

---

## New Tool: `cmd/healthcheck/main.go`

The actual CLI tool that ties everything together.

```go
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/itprodirect/go-hello-world/internal/checker"
	"github.com/itprodirect/go-hello-world/internal/workerpool"
)

func main() {
	targetsFile := flag.String("targets", "", "path to targets JSON file")
	workers := flag.Int("workers", 8, "number of concurrent workers")
	timeout := flag.Int("timeout", 5000, "default timeout per check in ms")
	jsonOutput := flag.Bool("json", false, "output results as JSON lines")
	flag.Parse()

	if *targetsFile == "" {
		// If no file provided, use some sensible defaults for demo
		fmt.Fprintln(os.Stderr, "Usage: healthcheck --targets targets.json")
		fmt.Fprintln(os.Stderr, "")
		fmt.Fprintln(os.Stderr, "No targets file provided. Using built-in demo targets.")
		fmt.Fprintln(os.Stderr, "")
	}

	var targets []checker.Target
	if *targetsFile != "" {
		var err error
		targets, err = checker.LoadTargets(*targetsFile)
		if err != nil {
			log.Fatalf("load targets: %v", err)
		}
	} else {
		targets = demoTargets()
	}

	// Apply default timeout to targets that don't specify one
	for i := range targets {
		if targets[i].Timeout <= 0 {
			targets[i].Timeout = *timeout
		}
	}

	// Context with cancellation on Ctrl+C
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	start := time.Now()
	pool := workerpool.New[checker.Target, checker.Result](*workers)
	results := pool.Run(ctx, targets, checker.Check)
	elapsed := time.Since(start)

	if *jsonOutput {
		for _, r := range results {
			line, _ := json.Marshal(r)
			fmt.Println(string(line))
		}
	} else {
		printTable(results)
	}

	// Summary
	up, down, errCount := 0, 0, 0
	for _, r := range results {
		switch r.Status {
		case "up":
			up++
		case "down":
			down++
		default:
			errCount++
		}
	}

	fmt.Fprintf(os.Stderr, "\n--- %d checks in %s | %d up | %d down | %d errors ---\n",
		len(results), elapsed.Round(time.Millisecond), up, down, errCount)
}

func printTable(results []checker.Result) {
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "STATUS\tNAME\tTYPE\tTARGET\tLATENCY\tDETAIL")
	fmt.Fprintln(w, "------\t----\t----\t------\t-------\t------")

	for _, r := range results {
		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s",
			checker.StatusEmoji(r.Status),
			r.Name,
			r.Type,
			r.Target,
			r.Latency.Round(time.Millisecond),
			r.Detail,
		)
		if r.TLS != nil {
			fmt.Fprintf(w, " (TLS: %d days left)", r.TLS.DaysLeft)
		}
		fmt.Fprintln(w)
	}

	w.Flush()
}

func demoTargets() []checker.Target {
	return []checker.Target{
		{Name: "google", URL: "https://www.google.com", Type: "http"},
		{Name: "github", URL: "https://github.com", Type: "http"},
		{Name: "localhost-8080", Host: "localhost", Port: 8080, Type: "tcp"},
		{Name: "dns-google", Host: "dns.google", Type: "dns"},
	}
}
```

### Example targets file: `targets.example.json`

```json
[
    {
        "name": "itprodirect-main",
        "url": "https://itprodirect.com",
        "type": "http"
    },
    {
        "name": "github-repo",
        "url": "https://github.com/itprodirect/go-hello-world",
        "type": "http"
    },
    {
        "name": "google-dns",
        "host": "8.8.8.8",
        "port": 53,
        "type": "tcp"
    },
    {
        "name": "cloudflare-dns",
        "host": "1.1.1.1",
        "type": "dns",
        "timeout_ms": 3000
    },
    {
        "name": "api-endpoint",
        "url": "https://httpbin.org/get",
        "type": "http",
        "timeout_ms": 5000
    }
]
```

---

## Concepts Demonstrated

| Go Pattern | Python Equivalent | Why Go Wins Here |
|-----------|------------------|-----------------|
| Goroutines (~2KB each) | `asyncio.Task` / threads (~8MB each) | Check 1000s of targets simultaneously |
| Channels for fan-out/fan-in | `asyncio.Queue` / `Queue` | No locks needed, compile-time safety |
| `context.WithTimeout` | `asyncio.wait_for` | Propagates cancellation through call stack |
| `sync.WaitGroup` | `asyncio.gather` | Wait for all workers to finish |
| Generic `Pool[In, Out]` | `concurrent.futures.Executor` | Type-safe, reusable for any batch job |
| `net.Dialer` | `socket.connect` | Built into stdlib, context-aware |
| `crypto/tls` | `ssl` module | TLS cert inspection in ~10 lines |
| Single binary output | `pip install` + venv + deps | `go build` → ship one file |

---

## Verification

```bash
go test ./internal/workerpool/...
go test ./internal/checker/...
go test ./...

# Run with demo targets (no file needed)
go run ./cmd/healthcheck

# Run with targets file
go run ./cmd/healthcheck --targets targets.example.json --workers 4

# JSON output for piping to Python
go run ./cmd/healthcheck --json | python3 -c "
import sys, json
for line in sys.stdin:
    r = json.loads(line)
    print(f\"{r['name']:20s} {r['status']:6s} {r['latency_ms']}ms\")
"

# Build single binary
go build -o bin/healthcheck ./cmd/healthcheck
./bin/healthcheck --targets targets.example.json
```
