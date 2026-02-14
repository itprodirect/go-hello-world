# Go Learning Roadmap: From Python/JS to Idiomatic Go

This roadmap extends the `go-hello-world` repo with five progressive phases.
Each phase adds a new `cmd/` entrypoint and/or `internal/` package, keeps
everything buildable and testable, and maps Go idioms back to Python/JS
equivalents for easier mental model transfer.

## Current State

The repo already covers:
- CLI vs HTTP server (`cmd/hello-cli`, `cmd/hello-server`)
- Goroutines + channels (worker pool in CLI)
- `sync.Mutex` (thread-safe counters)
- Basic `go test` with assertions

## Phase Overview

| Phase | Focus | New Packages / Commands | Key Go Concepts |
|-------|-------|------------------------|----------------|
| 1 | Error Handling | `internal/validator` | Custom errors, `errors.Is`/`As`, `%w` wrapping |
| 2 | Interfaces | `internal/greeter` refactor | Implicit satisfaction, dependency injection, composition |
| 3 | Table-Driven Tests & Benchmarks | Test refactors + benchmarks | `t.Run` subtests, `testing.B`, coverage |
| 4 | File I/O & Streams | `cmd/hello-transform`, `internal/transform` | `io.Reader`/`io.Writer`, `bufio`, `os` |
| 5 | Generics & Data Structures | `internal/collections` | Type parameters, constraints, generic utilities |

## How to Use These Docs with Codex

Each `PHASE-*.md` file is a self-contained implementation spec. To execute:

```
codex "Read docs/PHASE-01-errors.md and implement everything described.
       Run make test to verify. Commit when green."
```

Phases are independent — you can implement them in any order. Each doc
specifies exact file paths, function signatures, test expectations, and
Makefile updates.

## Python/JS Mental Model Map

| Python / JS | Go Equivalent | Phase |
|-------------|--------------|-------|
| `try/except`/`try/catch` | Multiple return values `(val, err)` | 1 |
| Duck typing | Implicit interface satisfaction | 2 |
| `pytest.mark.parametrize` / `describe/it` | Table-driven tests with `t.Run` | 3 |
| File streams / readline | `io.Reader`, `bufio.Scanner` | 4 |
| TypeScript generics / Python `TypeVar` | Go type parameters `[T any]` | 5 |

## Verification

After all phases, these commands should pass cleanly:

```bash
make fmt
make test
make build
go vet ./...
```
