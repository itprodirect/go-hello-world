# Phase 3: Table-Driven Tests & Benchmarks

## Why This Matters (Python/JS → Go)

In Python you use `@pytest.mark.parametrize`. In JS you use `describe/it`
with loops or `test.each`. In Go, the standard pattern is **table-driven
tests** — an array of test cases iterated with `t.Run()` subtests.

This phase refactors all existing tests to table-driven style, adds edge
case coverage, introduces benchmarks, and wires coverage into the Makefile.

## What to Build

### 1. Refactor: `internal/greeter/greeter_test.go`

Replace individual test functions with a single table-driven function:

```go
package greeter

import "testing"

func TestBuildGreeting(t *testing.T) {
    tests := []struct {
        name     string // test case label
        input    string
        sequence int
        want     string
    }{
        {"with name and sequence", "Nick", 3, "Hello, Nick! (#3)"},
        {"blank falls back to world", "   ", 0, "Hello, world!"},
        {"empty string falls back", "", 1, "Hello, world! (#1)"},
        {"trims whitespace", "  Alice  ", 2, "Hello, Alice! (#2)"},
        {"zero sequence omits number", "Bob", 0, "Hello, Bob!"},
        {"negative sequence omits number", "Eve", -1, "Hello, Eve!"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got := BuildGreeting(tt.input, tt.sequence)
            if got != tt.want {
                t.Errorf("BuildGreeting(%q, %d) = %q, want %q",
                    tt.input, tt.sequence, got, tt.want)
            }
        })
    }
}

func BenchmarkBuildGreeting(b *testing.B) {
    for i := 0; i < b.N; i++ {
        BuildGreeting("Nick", i+1)
    }
}
```

### 2. Refactor: `internal/metrics/counters_test.go`

```go
package metrics

import (
    "sync"
    "testing"
)

func TestCounters(t *testing.T) {
    tests := []struct {
        name   string
        ops    func(c *Counters)
        key    string
        want   uint64
    }{
        {
            name: "single inc",
            ops:  func(c *Counters) { c.Inc("hits") },
            key:  "hits",
            want: 1,
        },
        {
            name: "inc then add",
            ops: func(c *Counters) {
                c.Inc("hits")
                c.Add("hits", 4)
            },
            key:  "hits",
            want: 5,
        },
        {
            name: "unset key returns zero",
            ops:  func(c *Counters) {},
            key:  "missing",
            want: 0,
        },
        {
            name: "name normalization",
            ops:  func(c *Counters) { c.Inc("Hello World!") },
            key:  "hello_world_",
            want: 1,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            c := NewCounters()
            tt.ops(c)
            got := c.Get(tt.key)
            if got != tt.want {
                t.Errorf("Get(%q) = %d, want %d", tt.key, got, tt.want)
            }
        })
    }
}

func TestCountersConcurrent(t *testing.T) {
    c := NewCounters()
    const workers = 500
    var wg sync.WaitGroup
    wg.Add(workers)

    for i := 0; i < workers; i++ {
        go func() {
            defer wg.Done()
            c.Inc("shared")
        }()
    }
    wg.Wait()

    got := c.Get("shared")
    if got != workers {
        t.Fatalf("concurrent Inc: got %d, want %d", got, workers)
    }
}

func TestPlainTextOutput(t *testing.T) {
    c := NewCounters()
    c.Add("beta", 2)
    c.Add("alpha", 1)

    got := c.PlainText()
    want := "alpha 1\nbeta 2\n"
    if got != want {
        t.Errorf("PlainText() = %q, want %q", got, want)
    }
}

func TestPlainTextEmpty(t *testing.T) {
    c := NewCounters()
    got := c.PlainText()
    want := "no_counters 0\n"
    if got != want {
        t.Errorf("PlainText() empty = %q, want %q", got, want)
    }
}

func BenchmarkCountersInc(b *testing.B) {
    c := NewCounters()
    for i := 0; i < b.N; i++ {
        c.Inc("bench_counter")
    }
}

func BenchmarkCountersConcurrentInc(b *testing.B) {
    c := NewCounters()
    b.RunParallel(func(pb *testing.PB) {
        for pb.Next() {
            c.Inc("parallel")
        }
    })
}
```

### 3. Add validator tests (if Phase 1 is done)

Add table-driven tests in `internal/validator/validator_test.go`:

```go
func TestValidateName_Table(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr error // nil means valid
    }{
        {"valid name", "Nick", nil},
        {"empty", "", ErrEmpty},
        {"whitespace only", "   ", ErrEmpty},
        {"too long", strings.Repeat("a", 51), ErrTooLong},
        {"has angle bracket", "a<b", ErrBadChars},
        {"has ampersand", "a&b", ErrBadChars},
        {"max length ok", strings.Repeat("a", 50), nil},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateName(tt.input)
            if tt.wantErr == nil {
                if err != nil {
                    t.Fatalf("unexpected error: %v", err)
                }
                return
            }
            if !errors.Is(err, tt.wantErr) {
                t.Fatalf("got %v, want %v", err, tt.wantErr)
            }
        })
    }
}
```

### 4. Update Makefile

Add coverage and benchmark targets:

```makefile
.PHONY: fmt test test-cover bench build run-cli run-server

fmt:
	gofmt -w $(shell find . -name '*.go' -not -path './.git/*')

test:
	go test ./...

test-cover:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out
	@echo ""
	@echo "To view in browser: go tool cover -html=coverage.out"

bench:
	go test -bench=. -benchmem ./...

build:
	mkdir -p bin
	go build -o bin/hello-cli ./cmd/hello-cli
	go build -o bin/hello-server ./cmd/hello-server

run-cli:
	go run ./cmd/hello-cli --name Nick --repeat 3

run-server:
	go run ./cmd/hello-server
```

Add `coverage.out` to `.gitignore`.

## Concepts Demonstrated

| Concept | Python/JS Equivalent |
|---------|---------------------|
| `[]struct{...}` test table | `@pytest.mark.parametrize` / `test.each` |
| `t.Run("label", ...)` subtests | Named test cases with clear output |
| `t.Errorf` vs `t.Fatalf` | Soft vs hard assertion failure |
| `testing.B` benchmarks | `timeit` / `console.time` |
| `b.RunParallel` | Concurrent benchmark (no Python equivalent) |
| `-coverprofile` | `pytest-cov` / `nyc` |

## Verification

```bash
make test
make test-cover   # should show coverage percentages
make bench        # should show ns/op and allocs
```
