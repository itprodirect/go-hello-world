# Go Toolkit Roadmap

This repository is a production toolkit factory, not a learning workbook.

## Phase Status (As of March 5, 2026)

| Phase | Scope | Status |
|---|---|---|
| `PHASE-01-foundation.md` | `apperror`, `config`, CLI/server integration | Complete |
| `PHASE-02-interfaces.md` | `greeter` interface + `middleware` + CLI/server upgrades | Complete |
| `PHASE-03-healthcheck.md` | `workerpool`, `checker`, `cmd/healthcheck` | Complete (Hardened) |
| `PHASE-04-dataflow.md` | `pipeline`, `transform`, `cmd/dataflow` | Complete |
| `PHASE-05-generics.md` | `collections`, `cache` | Next |

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
- `cmd/dataflow`

Packages:
- `internal/apperror`
- `internal/config`
- `internal/greeter`
- `internal/middleware`
- `internal/metrics`
- `internal/workerpool`
- `internal/checker`
- `internal/validator`
- `internal/pipeline`
- `internal/transform`
- `internal/collections` (next)
- `internal/cache` (next)

## Completed Hardening Priorities

1. Healthcheck JSON contract + failure exit semantics.
2. Shared validator used by CLI and server paths.
3. Command-level test coverage for shipped binaries.
4. Phase docs cleanup to production-runbook format.
5. CI quality gates in `.github/workflows/ci.yml`.
6. Checker performance/timeout hardening (shared client + context-bound TLS probe).

## Next Build Target

Begin Phase 5 implementation (`internal/collections`, `internal/cache`).

## Quality Gate Policy

Every push/PR should pass:

- `gofmt` check
- `go vet ./...`
- `go test ./...`
- Coverage threshold (>= 70%)

CI workflow file:

- `.github/workflows/ci.yml`

## Session Notes

Detailed session history and verification outcomes live in `SESSION-LOG.md`.

## Legacy Docs

The `docs/` directory is archived learning-track material.
If there is any conflict, follow top-level phase docs.