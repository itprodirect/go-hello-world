# Phase 1: Error Handling Patterns

## Why This Matters (Python/JS → Go)

In Python you write `try/except`. In JS you write `try/catch` or `.catch()`.
In Go, errors are **values** — functions return them explicitly and callers
check them immediately. There are no exceptions and no stack unwinding.

This phase adds a `validator` package that demonstrates every common error
pattern a Go beginner needs.

## What to Build

### 1. New package: `internal/validator/validator.go`

A small input validation library used by both CLI and server.

```go
package validator

import (
    "errors"
    "fmt"
    "strings"
)

// --- Sentinel errors (like Python's custom exception classes) ---

var (
    ErrEmpty    = errors.New("value must not be empty")
    ErrTooLong  = errors.New("value exceeds maximum length")
    ErrBadChars = errors.New("value contains disallowed characters")
)

// ValidationError wraps an underlying error with field context.
// This is Go's version of a custom exception class.
type ValidationError struct {
    Field string
    Err   error
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation failed on %q: %v", e.Field, e.Err)
}

// Unwrap lets errors.Is and errors.As look through the wrapper.
func (e *ValidationError) Unwrap() error {
    return e.Err
}

// ValidateName checks a name string and returns a *ValidationError on failure.
func ValidateName(name string) error {
    trimmed := strings.TrimSpace(name)

    if trimmed == "" {
        return &ValidationError{Field: "name", Err: ErrEmpty}
    }

    if len(trimmed) > 50 {
        return &ValidationError{Field: "name", Err: fmt.Errorf("%w: got %d chars", ErrTooLong, len(trimmed))}
    }

    for _, ch := range trimmed {
        if ch == '<' || ch == '>' || ch == '&' {
            return &ValidationError{Field: "name", Err: fmt.Errorf("%w: %q", ErrBadChars, string(ch))}
        }
    }

    return nil
}

// ValidateRepeat checks the repeat count.
func ValidateRepeat(n int) error {
    if n < 1 {
        return &ValidationError{
            Field: "repeat",
            Err:   fmt.Errorf("must be >= 1, got %d", n),
        }
    }
    if n > 1000 {
        return &ValidationError{
            Field: "repeat",
            Err:   fmt.Errorf("must be <= 1000, got %d", n),
        }
    }
    return nil
}
```

### 2. New test file: `internal/validator/validator_test.go`

```go
package validator

import (
    "errors"
    "testing"
)

func TestValidateName_EmptyReturnsErrEmpty(t *testing.T) {
    err := ValidateName("   ")
    if err == nil {
        t.Fatal("expected error for blank name")
    }
    if !errors.Is(err, ErrEmpty) {
        t.Fatalf("expected ErrEmpty, got: %v", err)
    }
}

func TestValidateName_TooLongReturnsErrTooLong(t *testing.T) {
    long := string(make([]byte, 51)) // 51 zero-bytes
    err := ValidateName("a" + long)
    if !errors.Is(err, ErrTooLong) {
        t.Fatalf("expected ErrTooLong, got: %v", err)
    }
}

func TestValidateName_BadCharsReturnsErrBadChars(t *testing.T) {
    err := ValidateName("Nick<script>")
    if !errors.Is(err, ErrBadChars) {
        t.Fatalf("expected ErrBadChars, got: %v", err)
    }
}

func TestValidateName_ValidReturnsNil(t *testing.T) {
    if err := ValidateName("Nick"); err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
}

func TestValidateName_ErrorsAs_GetsField(t *testing.T) {
    err := ValidateName("")
    var ve *ValidationError
    if !errors.As(err, &ve) {
        t.Fatal("expected ValidationError")
    }
    if ve.Field != "name" {
        t.Fatalf("Field = %q, want %q", ve.Field, "name")
    }
}

func TestValidateRepeat_OutOfRange(t *testing.T) {
    if err := ValidateRepeat(0); err == nil {
        t.Fatal("expected error for 0")
    }
    if err := ValidateRepeat(1001); err == nil {
        t.Fatal("expected error for 1001")
    }
}

func TestValidateRepeat_ValidRange(t *testing.T) {
    for _, n := range []int{1, 500, 1000} {
        if err := ValidateRepeat(n); err != nil {
            t.Fatalf("unexpected error for %d: %v", n, err)
        }
    }
}
```

### 3. Integration: Update `cmd/hello-cli/main.go`

Add validation before the worker pool runs:

```go
// After flag.Parse(), before creating counters:
if err := validator.ValidateName(*name); err != nil {
    log.Fatalf("invalid input: %v", err)
}
if err := validator.ValidateRepeat(*repeat); err != nil {
    log.Fatalf("invalid input: %v", err)
}
```

Add the import: `"github.com/itprodirect/go-hello-world/internal/validator"`

### 4. Integration: Update `cmd/hello-server/main.go`

In the `/hello` handler, validate the name query param and return 400 on failure:

```go
name := r.URL.Query().Get("name")
if err := validator.ValidateName(name); err != nil {
    var ve *validator.ValidationError
    if errors.As(err, &ve) {
        http.Error(w, ve.Error(), http.StatusBadRequest)
        return
    }
    http.Error(w, "invalid input", http.StatusBadRequest)
    return
}
```

Note: skip validation when name is empty (keep the "world" fallback behavior).
Only validate when a name is actually provided.

## Concepts Demonstrated

| Concept | What It Replaces |
|---------|-----------------|
| `errors.New()` sentinel | `class MyError(Exception)` in Python |
| `fmt.Errorf("%w", ...)` wrapping | Chained exceptions in Python 3 |
| `errors.Is(err, target)` | `except MyError` matching |
| `errors.As(err, &target)` | `except MyError as e` with access to fields |
| Custom struct implementing `error` | Subclassing `Exception` |
| `Unwrap()` method | `__cause__` in Python |

## Verification

```bash
go test ./internal/validator/...
go run ./cmd/hello-cli --name "" --repeat 1     # should fail with validation error
go run ./cmd/hello-cli --name "Nick" --repeat 3  # should work as before
```
