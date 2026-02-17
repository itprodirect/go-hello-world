# Phase 2: Interfaces + HTTP Middleware Stack

> **Produces:** `internal/middleware`, greeter refactored to interface
> **Upgrades:** `cmd/hello-server`, `cmd/hello-cli`
> **Teaches:** Implicit interfaces, `http.Handler` composition, dependency injection, strategy pattern

## Implementation Status (February 17, 2026)

- Status: Complete
- Implemented:
  - `internal/greeter/greeter.go`
  - `internal/greeter/greeter_test.go`
  - `internal/middleware/middleware.go`
  - `internal/middleware/middleware_test.go`
  - `--style` support in `cmd/hello-cli/main.go`
  - middleware chain + style query support in `cmd/hello-server/main.go`
- Verification: included in full-repo `go test ./...`, `go vet ./...`, and `go build ./...` pass.

## Why This Phase

Go interfaces are the unlock. Once you understand that `http.Handler` is just
a one-method interface, and that middleware is just a function that wraps one
handler in another, you can build production HTTP services with zero frameworks.

This phase also refactors the greeter to demonstrate Go's most important
pattern: **implicit interface satisfaction** (compile-time duck typing).

---

## Package 1: Greeter Refactor — `internal/greeter/greeter.go`

Replace the single function with an interface + multiple implementations.
The old `BuildGreeting` function stays for backward compatibility.

```go
package greeter

import (
	"fmt"
	"strings"
)

// Greeter is the interface any greeting strategy must satisfy.
// In Python: a Protocol with a greet() method.
// In Go: any struct with this method signature auto-satisfies it.
type Greeter interface {
	Greet(name string, sequence int) string
}

// --- Implementations ---

// Standard is the default greeter (preserves original behavior).
type Standard struct{}

func (g Standard) Greet(name string, sequence int) string {
	return buildMsg(name, sequence, "Hello, %s! (#%d)", "Hello, %s!")
}

// Formal greets politely.
type Formal struct{}

func (g Formal) Greet(name string, sequence int) string {
	return buildMsg(name, sequence, "Good day, %s. [#%d]", "Good day, %s.")
}

// Shout greets loudly — useful for testing that the interface works with any impl.
type Shout struct{}

func (g Shout) Greet(name string, sequence int) string {
	base := buildMsg(name, sequence, "HEY %s!!! (#%d)", "HEY %s!!!")
	return strings.ToUpper(base)
}

// --- Factory ---

// New returns a Greeter by style name. Unknown styles get Standard.
func New(style string) Greeter {
	switch strings.ToLower(strings.TrimSpace(style)) {
	case "formal":
		return Formal{}
	case "shout":
		return Shout{}
	default:
		return Standard{}
	}
}

// BuildGreeting preserves the original API. Existing callers don't break.
func BuildGreeting(name string, sequence int) string {
	return Standard{}.Greet(name, sequence)
}

// --- shared ---

func buildMsg(name string, sequence int, withSeq, withoutSeq string) string {
	clean := strings.TrimSpace(name)
	if clean == "" {
		clean = "world"
	}
	if sequence > 0 {
		return fmt.Sprintf(withSeq, clean, sequence)
	}
	return fmt.Sprintf(withoutSeq, clean)
}
```

### Updated test: `internal/greeter/greeter_test.go`

```go
package greeter

import (
	"strings"
	"testing"
)

// Original tests still pass (backward compat).
func TestBuildGreetingWithSequence(t *testing.T) {
	got := BuildGreeting("Nick", 3)
	want := "Hello, Nick! (#3)"
	if got != want {
		t.Fatalf("BuildGreeting() = %q, want %q", got, want)
	}
}

func TestBuildGreetingFallsBackToWorld(t *testing.T) {
	got := BuildGreeting("   ", 0)
	want := "Hello, world!"
	if got != want {
		t.Fatalf("BuildGreeting() = %q, want %q", got, want)
	}
}

// Interface tests: every style satisfies Greeter.
func TestGreeterStyles(t *testing.T) {
	tests := []struct {
		style    string
		name     string
		sequence int
		contains string
	}{
		{"standard", "Nick", 1, "Hello, Nick!"},
		{"formal", "Nick", 1, "Good day, Nick."},
		{"shout", "Nick", 1, "HEY NICK!!!"},
		{"unknown", "Nick", 1, "Hello, Nick!"},
		{"standard", "", 0, "Hello, world!"},
		{"formal", "  ", 0, "Good day, world."},
	}

	for _, tt := range tests {
		t.Run(tt.style+"_"+tt.name, func(t *testing.T) {
			g := New(tt.style)
			got := g.Greet(tt.name, tt.sequence)
			if !strings.Contains(got, tt.contains) {
				t.Errorf("New(%q).Greet(%q, %d) = %q, want it to contain %q",
					tt.style, tt.name, tt.sequence, got, tt.contains)
			}
		})
	}
}

// Compile-time interface check.
var (
	_ Greeter = Standard{}
	_ Greeter = Formal{}
	_ Greeter = Shout{}
)
```

---

## Package 2: `internal/middleware/middleware.go`

Composable HTTP middleware — the pattern that makes Go HTTP servers powerful
without frameworks. Each middleware is a function that takes a handler and
returns a new handler. They chain like decorators in Python or Express
middleware in Node.

```go
package middleware

import (
	"log"
	"net/http"
	"runtime/debug"
	"strings"
	"time"

	"github.com/itprodirect/go-hello-world/internal/metrics"
)

// --- Logging ---

// Logger logs method, path, status, and duration for every request.
// Python equivalent: a WSGI/ASGI middleware or Flask's after_request.
func Logger(logger *log.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(sw, r)

		logger.Printf("%s %s %d %s",
			r.Method, r.URL.Path, sw.status, time.Since(start).Round(time.Microsecond))
	})
}

// statusWriter captures the status code for logging.
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

// --- Recovery (panic catcher) ---

// Recover catches panics in handlers and returns 500 instead of crashing.
// Python equivalent: try/except at the WSGI layer.
func Recover(logger *log.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				logger.Printf("PANIC: %v\n%s", err, debug.Stack())
				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// --- Request Counter ---

// RequestCounter increments a counter for every request by path.
// Uses the existing metrics.Counters package (reusing our legos).
func RequestCounter(counters *metrics.Counters, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		counters.Inc("http_requests_total")

		path := strings.Trim(r.URL.Path, "/")
		if path == "" {
			path = "root"
		}
		counters.Inc("path_" + path + "_requests")

		next.ServeHTTP(w, r)
	})
}

// --- Method Filter ---

// AllowMethods returns 405 for non-allowed HTTP methods.
// Cleans up handler code — no more `if r.Method != "GET"` boilerplate.
func AllowMethods(methods []string, next http.Handler) http.Handler {
	allowed := make(map[string]bool, len(methods))
	for _, m := range methods {
		allowed[strings.ToUpper(m)] = true
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !allowed[r.Method] {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// --- Chain helper ---

// Chain applies middlewares in order: Chain(a, b, c)(handler) = a(b(c(handler)))
// This is like Express's app.use(a, b, c) or Python decorator stacking.
func Chain(handler http.Handler, middlewares ...func(http.Handler) http.Handler) http.Handler {
	// Apply in reverse so the first middleware listed is the outermost.
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}
```

### Test: `internal/middleware/middleware_test.go`

```go
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
		w.Write([]byte("ok"))
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
		t.Errorf("log line missing expected content: %q", logLine)
	}
}

func TestRecover(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(&buf, "", 0)

	handler := Recover(logger, panicHandler())

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	// This should NOT panic — Recover catches it.
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want 500", rec.Code)
	}
	if !strings.Contains(buf.String(), "PANIC") {
		t.Errorf("expected PANIC in log output")
	}
}

func TestRequestCounter(t *testing.T) {
	counters := metrics.NewCounters()
	handler := RequestCounter(counters, okHandler())

	req := httptest.NewRequest(http.MethodGet, "/hello", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/hello", nil))

	if got := counters.Get("http_requests_total"); got != 2 {
		t.Errorf("http_requests_total = %d, want 2", got)
	}
	if got := counters.Get("path_hello_requests"); got != 2 {
		t.Errorf("path_hello_requests = %d, want 2", got)
	}
}

func TestAllowMethods(t *testing.T) {
	handler := AllowMethods([]string{"GET"}, okHandler())

	// GET should work
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("GET status = %d, want 200", rec.Code)
	}

	// POST should be rejected
	req = httptest.NewRequest(http.MethodPost, "/", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusMethodNotAllowed {
		t.Errorf("POST status = %d, want 405", rec.Code)
	}
}

func TestChain(t *testing.T) {
	counters := metrics.NewCounters()
	var buf bytes.Buffer
	logger := log.New(&buf, "", 0)

	// Stack: Logger → Recover → RequestCounter → handler
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
		t.Error("counter not incremented through chain")
	}
	if !strings.Contains(buf.String(), "GET") {
		t.Error("logger not invoked through chain")
	}
}
```

---

## Integration: Upgrade `cmd/hello-server/main.go`

Replace the inline `instrumentRequests` function with the middleware package.
Add `--style` flag support. The server handler setup becomes:

```go
import (
	"github.com/itprodirect/go-hello-world/internal/greeter"
	"github.com/itprodirect/go-hello-world/internal/middleware"
)

func main() {
	// ...existing setup...

	// Build the mux (same endpoints as before)
	mux := http.NewServeMux()

	mux.Handle("/hello", middleware.AllowMethods([]string{"GET"},
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			name := r.URL.Query().Get("name")
			style := r.URL.Query().Get("style")
			count := counters.Inc("hello_requests")

			g := greeter.New(style)
			message := g.Greet(name, int(count))

			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			json.NewEncoder(w).Encode(helloResponse{
				Message: message,
				Count:   count,
			})
		}),
	))

	// ...health and metrics handlers stay the same...

	// Stack middleware (replaces instrumentRequests)
	handler := middleware.Chain(
		mux,
		func(h http.Handler) http.Handler { return middleware.Logger(logger, h) },
		func(h http.Handler) http.Handler { return middleware.Recover(logger, h) },
		func(h http.Handler) http.Handler { return middleware.RequestCounter(counters, h) },
	)

	server := &http.Server{
		Addr:    ":8080",
		Handler: handler,
		// ...timeouts stay the same...
	}
}
```

### Upgrade `cmd/hello-cli/main.go`

Add `--style` flag:

```go
style := flag.String("style", "standard", "greeting style: standard, formal, shout")

// In the worker goroutine, replace:
//   message := greeter.BuildGreeting(*name, sequence)
// with:
g := greeter.New(*style)
message := g.Greet(*name, sequence)
```

**Remove** the old `instrumentRequests` function from the server — it's now
handled by the middleware chain.

---

## Concepts Demonstrated

| Go Pattern | Python/JS Equivalent | Real-World Use |
|-----------|---------------------|----------------|
| `Greeter` interface | ABC / Protocol / TS interface | Strategy pattern for pluggable behavior |
| Implicit satisfaction | Duck typing (but compile-checked) | No `implements` keyword needed |
| `New()` factory | `create_greeter("formal")` | Runtime selection of implementation |
| `http.Handler` interface | Express middleware / WSGI | Every HTTP framework is built on this |
| Middleware wrapping | `@decorator` / `app.use()` | Logging, auth, rate limiting, CORS |
| `Chain()` composer | `app.use(a, b, c)` | Declarative middleware ordering |
| `httptest.NewRequest` | `requests.mock` / `supertest` | Test HTTP handlers without starting server |

---

## Verification

```bash
go test ./internal/greeter/...
go test ./internal/middleware/...
go test ./...

# CLI with style flag
go run ./cmd/hello-cli --name Nick --repeat 3 --style formal
go run ./cmd/hello-cli --name Nick --repeat 2 --style shout

# Server with style query param
go run ./cmd/hello-server &
curl "http://localhost:8080/hello?name=Nick&style=formal"
curl "http://localhost:8080/hello?name=Nick&style=shout"
curl -X POST "http://localhost:8080/hello"  # should return 405
kill %1
```
