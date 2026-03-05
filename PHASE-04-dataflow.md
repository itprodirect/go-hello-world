# Phase 4: Dataflow Toolkit (Production)

> Produces: `internal/pipeline`, `internal/transform`, `cmd/dataflow`
> Ships: A Unix-style stream processor for high-throughput line transforms

## Implementation Status (March 5, 2026)

- Status: Complete
- Phase 4 deliverables:
  - `internal/pipeline/pipeline.go`
  - `internal/pipeline/pipeline_test.go`
  - `internal/transform/transform.go`
  - `internal/transform/transform_test.go`
  - `cmd/dataflow/main.go`
  - `cmd/dataflow/main_test.go`

## Production Contract

This phase uses contract-focused docs to avoid source drift. The files above are canonical.

### Input/Output

- Input source:
  - stdin (default)
  - file path via `--in`
- Output sink:
  - stdout (default)
  - file path via `--out`
- Data model:
  - line-based text processing (one output line per accepted input line)

### Modes

Supported `--mode` values:

- `upper`
- `lower`
- `trim`
- `number`
- `grep` (alias: `filter`) requires `--match`
- `drop` (alias: `exclude`) requires `--match`
- `json-field` (alias: `json-extract`) requires `--field`
- `json-pretty`
- `replace` requires `--old` (optional `--new`)
- `dedup`
- `chain` (trim -> dedup -> number)

### Concurrency

- `--workers 0`: sequential pipeline (`Run`)
- `--workers > 0`: concurrent pipeline (`RunConcurrent`)

Concurrent mode is cancellation-aware and intentionally does not preserve output order.

### Exit Codes

- `0`: success
- `1`: invalid mode/options, IO/pipeline failure, or runtime validation failure
- `2`: flag parse error

## Operational Notes

- `internal/pipeline` uses buffered scanning with increased token size (1 MB) for larger log lines.
- `RunConcurrent` handles cancellation and write errors without leaking worker goroutines.
- `cmd/dataflow` supports deterministic testing via `run` and `runWithContext` flow.

## Verification

Run from repo root:

```bash
go test ./...
go vet ./...
go test -cover ./...
```

Coverage snapshots for phase 4 artifacts:

- `cmd/dataflow`: 76.3%
- `internal/pipeline`: 91.7%
- `internal/transform`: 98.2%

## Examples

```bash
# Uppercase stdin
cat README.md | go run ./cmd/dataflow --mode upper

# Filter lines containing "error"
cat app.log | go run ./cmd/dataflow --mode filter --match "error"

# Extract JSON field values
cat events.jsonl | go run ./cmd/dataflow --mode json-extract --field status

# Use concurrent processing and write to file
go run ./cmd/dataflow --mode chain --workers 4 --in app.log --out cleaned.log
```