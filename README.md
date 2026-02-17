# Go Toolkit: Reusable Blocks to Real Tools

Production-focused Go toolkit factory.
Each phase adds reusable `internal/` packages and working `cmd/` binaries.

## Project Status (As of February 17, 2026)

| Phase | Status |
|---|---|
| `PHASE-01-foundation.md` | Complete |
| `PHASE-02-interfaces.md` | Complete |
| `PHASE-03-healthcheck.md` | Complete |
| `PHASE-04-dataflow.md` | Next |
| `PHASE-05-generics.md` | Planned |

## Current Binaries

| Binary | Purpose | Status |
|---|---|---|
| `hello-cli` | Concurrent greeting generator | Ready |
| `hello-server` | HTTP API with middleware, metrics, graceful shutdown | Ready |
| `healthcheck` | Concurrent HTTP/TCP/DNS endpoint checker | Ready |
| `dataflow` | Stream processor for pipelines | Planned |

## Current Packages

| Package | Purpose | Status |
|---|---|---|
| `internal/greeter` | Greeting strategies via interface | Ready |
| `internal/metrics` | Thread-safe in-memory counters | Ready |
| `internal/apperror` | Structured errors, wrapping, sentinels | Ready |
| `internal/config` | JSON config + env var overrides | Ready |
| `internal/middleware` | HTTP logging/recovery/method/counter middleware | Ready |
| `internal/workerpool` | Generic concurrent fan-out/fan-in | Ready |
| `internal/checker` | HTTP/TCP/DNS checks with TLS details | Ready |
| `internal/pipeline` | Stream processing engine | Planned |
| `internal/transform` | Text and JSON transforms | Planned |
| `internal/collections` | Generic collection helpers | Planned |
| `internal/cache` | Generic TTL cache | Planned |

## Canonical Roadmap

Use top-level production docs as source of truth:

- `ROADMAP.md`
- `PHASE-01-foundation.md`
- `PHASE-02-interfaces.md`
- `PHASE-03-healthcheck.md`
- `PHASE-04-dataflow.md`
- `PHASE-05-generics.md`

## Session History

- `SESSION-LOG.md`

## Quick Start

```bash
make test
make vet
make build
```

## CLI Examples

```bash
# Greeting CLI
go run ./cmd/hello-cli --name Nick --repeat 3 --style formal

# HTTP server
go run ./cmd/hello-server --config config.example.json
curl "http://localhost:8080/hello?name=Nick&style=shout"
curl "http://localhost:8080/metrics"

# Health checker
go run ./cmd/healthcheck --targets targets.example.json --workers 4
go run ./cmd/healthcheck --json
```

## Legacy Learning Docs

`docs/` is archived learning-track material.
For active implementation work, use top-level roadmap/phase files.
