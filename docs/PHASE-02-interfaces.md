# Phase 2: Interfaces & Dependency Injection

## Why This Matters (Python/JS → Go)

Python uses duck typing: if it has a `.quack()` method, it's a duck.
Go does the same thing, but **at compile time**. You never write
`class Foo implements Bar` — if your struct has the right methods,
it satisfies the interface automatically.

This phase refactors `greeter` to use an interface, adds multiple
greeting styles, and shows how to inject different implementations.

## What to Build

### 1. Refactor: `internal/greeter/greeter.go`

Define a `Greeter` interface and make the existing logic one implementation:

```go
package greeter

import (
    "fmt"
    "strings"
)

// Greeter is the interface any greeting strategy must satisfy.
// In Python terms: any object with a .Greet(name, seq) method works.
type Greeter interface {
    Greet(name string, sequence int) string
}

// --- Standard greeter (current behavior) ---

type StandardGreeter struct{}

func (g StandardGreeter) Greet(name string, sequence int) string {
    return buildMessage(name, sequence, "Hello, %s! (#%d)", "Hello, %s!")
}

// --- Formal greeter ---

type FormalGreeter struct{}

func (g FormalGreeter) Greet(name string, sequence int) string {
    return buildMessage(name, sequence, "Good day, %s. [#%d]", "Good day, %s.")
}

// --- Casual greeter ---

type CasualGreeter struct{}

func (g CasualGreeter) Greet(name string, sequence int) string {
    return buildMessage(name, sequence, "Hey %s! (#%d)", "Hey %s!")
}

// BuildGreeting preserves the original public API for backward compatibility.
func BuildGreeting(name string, sequence int) string {
    return StandardGreeter{}.Greet(name, sequence)
}

// NewGreeter returns a Greeter by style name. Unknown styles fall back to standard.
func NewGreeter(style string) Greeter {
    switch strings.ToLower(strings.TrimSpace(style)) {
    case "formal":
        return FormalGreeter{}
    case "casual":
        return CasualGreeter{}
    default:
        return StandardGreeter{}
    }
}

// --- shared helper ---

func buildMessage(name string, sequence int, withSeq, withoutSeq string) string {
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

### 2. Update tests: `internal/greeter/greeter_test.go`

Keep existing tests passing and add interface-level tests:

```go
package greeter

import "testing"

// Existing tests stay unchanged — BuildGreeting still works.

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

// New: test that each style satisfies the Greeter interface.
func TestGreeterInterface(t *testing.T) {
    tests := []struct {
        style    string
        name     string
        sequence int
        contains string
    }{
        {"standard", "Nick", 1, "Hello, Nick!"},
        {"formal", "Nick", 1, "Good day, Nick."},
        {"casual", "Nick", 1, "Hey Nick!"},
        {"unknown", "Nick", 1, "Hello, Nick!"},  // fallback
    }

    for _, tt := range tests {
        t.Run(tt.style, func(t *testing.T) {
            g := NewGreeter(tt.style)
            got := g.Greet(tt.name, tt.sequence)
            if got == "" || !containsSubstring(got, tt.contains) {
                t.Fatalf("NewGreeter(%q).Greet() = %q, want it to contain %q", tt.style, got, tt.contains)
            }
        })
    }
}

func containsSubstring(s, sub string) bool {
    return len(s) >= len(sub) && (s == sub || len(sub) == 0 || findSubstring(s, sub))
}

func findSubstring(s, sub string) bool {
    for i := 0; i <= len(s)-len(sub); i++ {
        if s[i:i+len(sub)] == sub {
            return true
        }
    }
    return false
}
```

**Note:** You can simplify `containsSubstring` by importing `"strings"` and
using `strings.Contains`. The manual version above avoids the import for
demonstration, but `strings.Contains` is idiomatic.

### 3. Update CLI: `cmd/hello-cli/main.go`

Add a `--style` flag:

```go
style := flag.String("style", "standard", "greeting style: standard, formal, casual")
```

Replace the direct `greeter.BuildGreeting` call in the worker with:

```go
g := greeter.NewGreeter(*style)

// Inside the worker goroutine:
message := g.Greet(*name, sequence)
```

### 4. Update Server: `cmd/hello-server/main.go`

Read style from query param `?style=formal`:

```go
style := r.URL.Query().Get("style")
g := greeter.NewGreeter(style)
message := g.Greet(name, int(count))
```

## Concepts Demonstrated

| Concept | Python/JS Equivalent |
|---------|---------------------|
| `Greeter` interface with `Greet()` method | ABC / Protocol in Python, TS interface |
| Implicit satisfaction (no `implements`) | Python duck typing, but compile-checked |
| `NewGreeter()` factory function | Factory pattern / class selection |
| Interface as function parameter | Dependency injection |
| Zero-value structs as implementations | Stateless strategy pattern |

## Verification

```bash
go test ./internal/greeter/...
go run ./cmd/hello-cli --name Nick --repeat 3 --style formal
go run ./cmd/hello-cli --name Nick --repeat 3 --style casual
curl "http://localhost:8080/hello?name=Nick&style=formal"
```
