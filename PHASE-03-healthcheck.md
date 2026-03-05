# Phase 3: Healthcheck Toolkit (Production)

> Produces: `internal/checker`, `internal/workerpool`, `cmd/healthcheck`
> Ships: A concurrent HTTP/TCP/DNS health checker suitable for CI and ops checks

## Implementation Status (March 5, 2026)

- Status: Complete and hardened
- Phase 3 deliverables:
  - `internal/workerpool/workerpool.go`
  - `internal/workerpool/workerpool_test.go`
  - `internal/checker/checker.go`
  - `internal/checker/checker_test.go`
  - `cmd/healthcheck/main.go`
  - `cmd/healthcheck/main_test.go`
  - `targets.example.json`

## Production Contract

This phase intentionally avoids embedding full source listings.
The source files above are the canonical implementation.

### Input Contract

Targets are loaded from JSON (`--targets`) with this schema:

- `name` (string)
- `type` (string): `http`, `tcp`, `dns`
- `url` (string, http)
- `host` (string, tcp/dns)
- `port` (int, tcp)
- `timeout_ms` (int, optional per target)

If `--targets` is omitted, the CLI runs built-in demo targets.

### Output Contract

- Table mode: human-readable status table
- JSON mode (`--json`): one JSON object per result line
- `latency_ms` is emitted as integer milliseconds (not nanoseconds)

Example JSON result shape:

```json
{
  "name": "github",
  "type": "http",
  "target": "https://github.com",
  "status": "up",
  "latency_ms": 142,
  "detail": "HTTP 200"
}
```

### Exit Codes

- `0`: all checks are `up`
- `1`: runtime/validation/check failure, or any `down`/`error` result
- `2`: flag parse error

## Operational Notes

- Per-target timeout defaults to `--timeout` when `timeout_ms` is missing.
- Worker concurrency is controlled with `--workers`.
- Summary is printed to stderr in all modes.

## Verification

Run from repo root:

```bash
go test ./...
go vet ./...
go test -cover ./...
```

Current package coverage snapshots:

- `cmd/healthcheck`: 77.8%
- `internal/checker`: 73.2%
- `internal/workerpool`: 93.3%

## Next Hardening Targets

These are tracked after Phase 3 completion:

1. Reuse shared HTTP transport/client in checker for connection pooling.
2. Make TCP TLS probe fully context-bound to phase timeout budget.
3. Keep CLI and JSON contracts stable with command-level tests.
