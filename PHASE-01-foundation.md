# Phase 1: Foundation — Config + Structured Errors

> **Produces:** `internal/config`, `internal/apperror`
> **Upgrades:** `cmd/hello-cli`, `cmd/hello-server`
> **Teaches:** Error values, wrapping, `errors.Is/As`, JSON config, env var overrides

## Implementation Status (February 17, 2026)

- Status: Complete
- Implemented:
  - `internal/apperror/apperror.go`
  - `internal/apperror/apperror_test.go`
  - `internal/config/config.go`
  - `internal/config/config_test.go`
  - `config.example.json`
  - Integration updates in `cmd/hello-cli/main.go` and `cmd/hello-server/main.go`
- Verification: `go test ./...`, `go vet ./...`, and `go build ./...` all passed in this session.

## Why This Phase First

Every real tool needs two things before anything else: a way to load
configuration and a way to handle errors properly. In Python you'd use
`argparse` + `pydantic` + `try/except`. In Go, errors are values you pass
around explicitly, and config is just a struct you unmarshal into.

These two packages will be imported by **every future phase**.

---

## Package 1: `internal/apperror/apperror.go`

Custom error types with wrapping, sentinel errors, and field context.
This is the Go equivalent of Python's custom exception hierarchy.

```go
package apperror

import (
	"errors"
	"fmt"
)

// --- Sentinel errors (match with errors.Is) ---

var (
	ErrNotFound    = errors.New("not found")
	ErrValidation  = errors.New("validation failed")
	ErrTimeout     = errors.New("operation timed out")
	ErrUnavailable = errors.New("service unavailable")
)

// FieldError attaches a field name to any error.
// Go's version of a custom exception with structured context.
type FieldError struct {
	Field   string
	Message string
	Err     error
}

func (e *FieldError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Field, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// Unwrap enables errors.Is and errors.As to see through this wrapper.
func (e *FieldError) Unwrap() error {
	return e.Err
}

// NewFieldError creates a FieldError wrapping a sentinel.
func NewFieldError(field, message string, sentinel error) *FieldError {
	return &FieldError{
		Field:   field,
		Message: message,
		Err:     sentinel,
	}
}

// Wrap adds context to any error without losing the original.
// Python equivalent: raise NewError("context") from original_error
func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}

// IsNotFound is a convenience check.
func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

// IsValidation is a convenience check.
func IsValidation(err error) bool {
	return errors.Is(err, ErrValidation)
}

// IsTimeout is a convenience check.
func IsTimeout(err error) bool {
	return errors.Is(err, ErrTimeout)
}
```

### Test: `internal/apperror/apperror_test.go`

```go
package apperror

import (
	"errors"
	"testing"
)

func TestFieldError_ErrorString(t *testing.T) {
	tests := []struct {
		name  string
		fe    *FieldError
		want  string
	}{
		{
			name: "with sentinel",
			fe:   NewFieldError("name", "must not be empty", ErrValidation),
			want: "name: must not be empty: validation failed",
		},
		{
			name: "without sentinel",
			fe:   &FieldError{Field: "age", Message: "required"},
			want: "age: required",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fe.Error()
			if got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFieldError_Unwrap(t *testing.T) {
	fe := NewFieldError("url", "bad format", ErrValidation)

	if !errors.Is(fe, ErrValidation) {
		t.Fatal("errors.Is should match ErrValidation through FieldError")
	}

	var target *FieldError
	if !errors.As(fe, &target) {
		t.Fatal("errors.As should extract *FieldError")
	}
	if target.Field != "url" {
		t.Errorf("Field = %q, want %q", target.Field, "url")
	}
}

func TestWrap_NilPassthrough(t *testing.T) {
	if got := Wrap(nil, "context"); got != nil {
		t.Fatalf("Wrap(nil) should return nil, got %v", got)
	}
}

func TestWrap_PreservesSentinel(t *testing.T) {
	original := ErrTimeout
	wrapped := Wrap(original, "checking endpoint")
	if !errors.Is(wrapped, ErrTimeout) {
		t.Fatal("wrapped error should still match ErrTimeout")
	}
}

func TestConvenienceChecks(t *testing.T) {
	tests := []struct {
		name  string
		err   error
		check func(error) bool
		want  bool
	}{
		{"IsNotFound true", ErrNotFound, IsNotFound, true},
		{"IsNotFound false", ErrValidation, IsNotFound, false},
		{"IsValidation true", NewFieldError("x", "y", ErrValidation), IsValidation, true},
		{"IsTimeout true", Wrap(ErrTimeout, "ctx"), IsTimeout, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.check(tt.err); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
```

---

## Package 2: `internal/config/config.go`

Load configuration from a JSON file with environment variable overrides.
This is the Go equivalent of Python's `pydantic` Settings or `dotenv`.

```go
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// AppConfig holds all application-level settings.
// Each field maps to a JSON key and an env var override.
type AppConfig struct {
	// Server settings
	Host string `json:"host"`
	Port int    `json:"port"`

	// Application settings
	Name         string `json:"name"`
	DefaultGreet string `json:"default_greet"`
	LogLevel     string `json:"log_level"`

	// Feature flags
	JSONOutput bool `json:"json_output"`
}

// DefaultConfig returns sensible defaults (like a Python dataclass with defaults).
func DefaultConfig() AppConfig {
	return AppConfig{
		Host:         "0.0.0.0",
		Port:         8080,
		Name:         "go-hello-world",
		DefaultGreet: "world",
		LogLevel:     "info",
		JSONOutput:   false,
	}
}

// Load reads config from a JSON file path, then applies env var overrides.
// Missing file is not an error — you just get defaults + env vars.
func Load(path string) (AppConfig, error) {
	cfg := DefaultConfig()

	if path != "" {
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				// No file is fine — use defaults + env vars
				applyEnvOverrides(&cfg)
				return cfg, nil
			}
			return cfg, fmt.Errorf("read config %s: %w", path, err)
		}

		if err := json.Unmarshal(data, &cfg); err != nil {
			return cfg, fmt.Errorf("parse config %s: %w", path, err)
		}
	}

	applyEnvOverrides(&cfg)
	return cfg, nil
}

// MustLoad calls Load and panics on error. Use in main() where you'd os.Exit anyway.
func MustLoad(path string) AppConfig {
	cfg, err := Load(path)
	if err != nil {
		panic(fmt.Sprintf("config: %v", err))
	}
	return cfg
}

// applyEnvOverrides checks for APP_* environment variables.
// Env vars always win over file values (12-factor app style).
func applyEnvOverrides(cfg *AppConfig) {
	if v := os.Getenv("APP_HOST"); v != "" {
		cfg.Host = v
	}
	if v := os.Getenv("APP_PORT"); v != "" {
		if port, err := strconv.Atoi(v); err == nil {
			cfg.Port = port
		}
	}
	if v := os.Getenv("APP_NAME"); v != "" {
		cfg.Name = v
	}
	if v := os.Getenv("APP_DEFAULT_GREET"); v != "" {
		cfg.DefaultGreet = v
	}
	if v := os.Getenv("APP_LOG_LEVEL"); v != "" {
		cfg.LogLevel = strings.ToLower(v)
	}
	if v := os.Getenv("APP_JSON_OUTPUT"); v != "" {
		cfg.JSONOutput = v == "true" || v == "1"
	}
}

// Addr returns "host:port" for http.Server.
func (c AppConfig) Addr() string {
	return fmt.Sprintf("%s:%d", c.Host, c.Port)
}
```

### Test: `internal/config/config_test.go`

```go
package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want 8080", cfg.Port)
	}
	if cfg.DefaultGreet != "world" {
		t.Errorf("DefaultGreet = %q, want %q", cfg.DefaultGreet, "world")
	}
	if cfg.LogLevel != "info" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "info")
	}
}

func TestLoad_MissingFileReturnsDefaults(t *testing.T) {
	cfg, err := Load("/nonexistent/path.json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want default 8080", cfg.Port)
	}
}

func TestLoad_FromJSONFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	data := []byte(`{"port": 9090, "name": "test-app", "log_level": "debug"}`)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Port != 9090 {
		t.Errorf("Port = %d, want 9090", cfg.Port)
	}
	if cfg.Name != "test-app" {
		t.Errorf("Name = %q, want %q", cfg.Name, "test-app")
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want %q", cfg.LogLevel, "debug")
	}
	// Fields not in JSON keep defaults
	if cfg.DefaultGreet != "world" {
		t.Errorf("DefaultGreet = %q, want default %q", cfg.DefaultGreet, "world")
	}
}

func TestLoad_EnvOverridesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")

	data := []byte(`{"port": 9090}`)
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}

	t.Setenv("APP_PORT", "3000")
	t.Setenv("APP_LOG_LEVEL", "DEBUG")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Env var wins over file
	if cfg.Port != 3000 {
		t.Errorf("Port = %d, want 3000 (env override)", cfg.Port)
	}
	if cfg.LogLevel != "debug" {
		t.Errorf("LogLevel = %q, want %q (lowercased env)", cfg.LogLevel, "debug")
	}
}

func TestLoad_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")

	if err := os.WriteFile(path, []byte(`{not json}`), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestAddr(t *testing.T) {
	cfg := AppConfig{Host: "localhost", Port: 3000}
	got := cfg.Addr()
	want := "localhost:3000"
	if got != want {
		t.Errorf("Addr() = %q, want %q", got, want)
	}
}
```

---

## Integration: Upgrade Existing Tools

### `cmd/hello-cli/main.go` changes

Add validation using `apperror`:

```go
import "github.com/itprodirect/go-hello-world/internal/apperror"

// After flag.Parse():
if *name != "" {
    for _, ch := range *name {
        if ch == '<' || ch == '>' || ch == '&' {
            log.Fatalf("invalid input: %v",
                apperror.NewFieldError("name", "contains unsafe characters", apperror.ErrValidation))
        }
    }
}
if *repeat < 1 || *repeat > 1000 {
    log.Fatalf("invalid input: %v",
        apperror.NewFieldError("repeat", "must be 1-1000", apperror.ErrValidation))
}
```

### `cmd/hello-server/main.go` changes

Add config loading and use `apperror` for request validation:

```go
import (
    "github.com/itprodirect/go-hello-world/internal/config"
    "github.com/itprodirect/go-hello-world/internal/apperror"
)

func main() {
    cfgPath := flag.String("config", "", "path to config JSON file")
    flag.Parse()

    cfg := config.MustLoad(*cfgPath)
    logger := log.New(os.Stdout, "", log.LstdFlags)
    logger.Printf("loaded config: %s (port %d)", cfg.Name, cfg.Port)

    // Use cfg.Addr() instead of hardcoded ":8080"
    server := &http.Server{
        Addr: cfg.Addr(),
        // ...
    }
}
```

In the `/hello` handler, validate with `apperror`:

```go
name := r.URL.Query().Get("name")
if name == "" {
    name = cfg.DefaultGreet
}
for _, ch := range name {
    if ch == '<' || ch == '>' || ch == '&' {
        ve := apperror.NewFieldError("name", "contains unsafe characters", apperror.ErrValidation)
        http.Error(w, ve.Error(), http.StatusBadRequest)
        return
    }
}
```

### Example config file: `config.example.json`

```json
{
    "host": "0.0.0.0",
    "port": 8080,
    "name": "go-hello-world",
    "default_greet": "world",
    "log_level": "info",
    "json_output": false
}
```

---

## Concepts Demonstrated

| Go Pattern | Python/JS Equivalent | Why It Matters |
|-----------|---------------------|----------------|
| `errors.New()` sentinels | `class NotFoundError(Exception)` | Match error types without strings |
| `fmt.Errorf("%w", err)` wrapping | `raise X from Y` / chained exceptions | Add context without losing original |
| `errors.Is(err, target)` | `except NotFoundError` | Check through any depth of wrapping |
| `errors.As(err, &target)` | `except MyError as e` | Extract structured error fields |
| `(value, error)` return | try/except blocks | Errors are explicit control flow |
| JSON struct tags | Pydantic model fields | Declarative config schema |
| `os.Getenv` overrides | `os.environ.get()` / `dotenv` | 12-factor app config |

---

## Verification

```bash
go test ./internal/apperror/...
go test ./internal/config/...
go test ./...
go run ./cmd/hello-cli --name "Nick" --repeat 3
go run ./cmd/hello-cli --name "Nick<script>" --repeat 1  # should fail
go run ./cmd/hello-server --config config.example.json
```
