# Go Beginner Demo: CLI vs HTTP Server

This repo demonstrates two common Go use-cases with shared code:
- A short-lived CLI command (`hello-cli`)
- A long-running HTTP service (`hello-server`)

Both programs share `internal/greeter` and `internal/metrics`, and both use concurrency in a beginner-friendly way.

## Project Structure

```text
.
├── cmd/
│   ├── hello-cli/main.go
│   └── hello-server/main.go
├── internal/
│   ├── greeter/
│   │   ├── greeter.go
│   │   └── greeter_test.go
│   └── metrics/
│       ├── counters.go
│       └── counters_test.go
├── .gitignore
├── go.mod
├── Makefile
└── README.md
```

## CLI Use-Case (`hello-cli`)

CLI programs are usually:
- Started by a user
- Do a task quickly
- Exit

Run it:

```bash
go run ./cmd/hello-cli --name Nick --repeat 3
```

Flags:
- `--name` (default `world`)
- `--repeat` (default `1`)
- `--json` (emit JSON lines)

Concurrency in CLI:
- A worker pool uses goroutines + channels to build greetings concurrently.
- Output order is preserved by storing each result by index before printing.

Example text output:

```text
Hello, Nick! (#1)
Hello, Nick! (#2)
Hello, Nick! (#3)
```

Example JSON lines output:

```bash
go run ./cmd/hello-cli --name Nick --repeat 3 --json
```

```json
{"index":1,"message":"Hello, Nick! (#1)"}
{"index":2,"message":"Hello, Nick! (#2)"}
{"index":3,"message":"Hello, Nick! (#3)"}
```

## Server Use-Case (`hello-server`)

Servers are usually:
- Started once
- Handle many requests over time
- Keep running until stopped

Run it:

```bash
go run ./cmd/hello-server
```

Endpoints:
- `GET /hello?name=Nick` -> JSON response with message + request count
- `GET /health` -> `200 OK` plain text
- `GET /metrics` -> plain text counters

Concurrency in server:
- Each HTTP request is handled concurrently by Go's `net/http`.
- A background goroutine logs an uptime tick every 5 seconds and increments a counter.

Server timeout settings are configured (`ReadHeaderTimeout`, `ReadTimeout`, `WriteTimeout`, `IdleTimeout`) for safer defaults.

Example:

```bash
curl "http://localhost:8080/hello?name=Nick"
```

```json
{"message":"Hello, Nick! (#1)","count":1}
```

## Shared Packages

- `internal/greeter`: builds greeting strings
- `internal/metrics`: thread-safe in-memory counters used by both CLI and server

## Makefile Commands

```bash
make fmt
make test
make build
make run-cli
make run-server
```

## Verify Locally

Run exactly these commands:

```bash
go test ./...
go run ./cmd/hello-cli --name Nick --repeat 3
go run ./cmd/hello-server
```

## What This Demo Highlights

- Go supports both one-off CLI tools and long-running services with minimal setup.
- `cmd/...` entrypoints keep different runtime styles clearly separated.
- Shared internal packages encourage reuse without external dependencies.
- Goroutines are lightweight and simple to start for concurrent work.
- Channels provide a clear pattern for coordinating concurrent workers.
- You can keep concurrency safe by controlling shared state through a mutex-backed package.
- `net/http` in the standard library is enough to build production-style handlers and routing basics.
- Timeouts on `http.Server` are straightforward and important for robust services.
- `go test` and small unit tests make behavior easy to verify early.
