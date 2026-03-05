# Go Toolkit Roadmap

This repository is a production toolkit factory, not a learning workbook.

## Phase Status (As of March 5, 2026)

| Phase | Scope | Status |
|---|---|---|
| `PHASE-01-foundation.md` | `apperror`, `config`, CLI/server integration | Complete |
| `PHASE-02-interfaces.md` | `greeter` interface + `middleware` + CLI/server upgrades | Complete |
| `PHASE-03-healthcheck.md` | `workerpool`, `checker`, `cmd/healthcheck` | Complete (Hardened) |
| `PHASE-04-dataflow.md` | `pipeline`, `transform`, `cmd/dataflow` | Next |
| `PHASE-05-generics.md` | `collections`, `cache` | Planned |

## Canonical Phase Docs

1. `PHASE-01-foundation.md`
2. `PHASE-02-interfaces.md`
3. `PHASE-03-healthcheck.md`
4. `PHASE-04-dataflow.md`
5. `PHASE-05-generics.md`

## Execution Guidance

- Treat top-level `PHASE-*.md` files as the source of truth.
- Keep the repo buildable after each phase.
- Run package-level tests as each package lands, then run full-repo checks.
- Preserve existing behavior unless a phase explicitly changes it.
- Prefer contract-focused docs over copied code blocks to avoid drift.

## Current Production Targets

Binaries:
- `cmd/hello-cli`
- `cmd/hello-server`
- `cmd/healthcheck`
- `cmd/dataflow` (pending)

Packages:
- `internal/apperror`
- `internal/config`
- `internal/greeter`
- `internal/middleware`
- `internal/metrics`
- `internal/workerpool`
- `internal/checker`
- `internal/validator`
- `internal/pipeline` (pending)
- `internal/transform` (pending)
- `internal/collections` (pending)
- `internal/cache` (pending)

## Current Hardening Focus

1. Priority 1 complete: healthcheck JSON contract + failure exit semantics.
2. Priority 2 complete: shared validator used by CLI and server paths.
3. Priority 3 complete: command-level test coverage for shipped binaries.
4. Priority 5 in progress: phase docs cleanup for production trust.
5. Priority 4 next: CI quality gates (`fmt`, `vet`, `test`, coverage threshold).

## Session Notes

Detailed session history and verification outcomes live in `SESSION-LOG.md`.

## Legacy Docs

The `docs/` directory is archived learning-track material.
If there is any conflict, follow top-level phase docs.
