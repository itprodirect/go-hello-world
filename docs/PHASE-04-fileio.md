# Phase 4: File I/O & the io.Reader/Writer Pattern

## Why This Matters (Python/JS → Go)

In Python you `open()` files and iterate lines. In Node you use streams or
`fs.readFile`. Go's superpower is `io.Reader` and `io.Writer` — tiny
interfaces that let you compose file I/O, HTTP bodies, buffers, compression,
and encryption with the same code. Once you understand these two interfaces,
you understand half of Go's standard library.

This phase adds a text transformation pipeline as a new CLI tool.

## What to Build

### 1. New package: `internal/transform/transform.go`

A composable text transformer that works on any `io.Reader` → `io.Writer`:

```go
package transform

import (
    "bufio"
    "fmt"
    "io"
    "strings"
)

// TransformFunc takes a line and returns a transformed line.
type TransformFunc func(line string) string

// Process reads lines from r, applies fn to each, and writes to w.
// This works with files, stdin, HTTP bodies, buffers — anything.
func Process(r io.Reader, w io.Writer, fn TransformFunc) error {
    scanner := bufio.NewScanner(r)
    for scanner.Scan() {
        transformed := fn(scanner.Text())
        if _, err := fmt.Fprintln(w, transformed); err != nil {
            return fmt.Errorf("write: %w", err)
        }
    }
    return scanner.Err()
}

// --- Built-in transform functions ---

// Upper converts each line to uppercase.
func Upper(line string) string {
    return strings.ToUpper(line)
}

// Lower converts each line to lowercase.
func Lower(line string) string {
    return strings.ToLower(line)
}

// NumberLines returns a TransformFunc that prefixes each line with its number.
func NumberLines() TransformFunc {
    n := 0
    return func(line string) string {
        n++
        return fmt.Sprintf("%4d | %s", n, line)
    }
}

// Chain composes multiple TransformFuncs left to right.
func Chain(fns ...TransformFunc) TransformFunc {
    return func(line string) string {
        for _, fn := range fns {
            line = fn(line)
        }
        return line
    }
}
```

### 2. Tests: `internal/transform/transform_test.go`

```go
package transform

import (
    "bytes"
    "strings"
    "testing"
)

func TestProcess(t *testing.T) {
    tests := []struct {
        name  string
        input string
        fn    TransformFunc
        want  string
    }{
        {
            name:  "uppercase",
            input: "hello\nworld\n",
            fn:    Upper,
            want:  "HELLO\nWORLD\n",
        },
        {
            name:  "lowercase",
            input: "HELLO\nWORLD\n",
            fn:    Lower,
            want:  "hello\nworld\n",
        },
        {
            name:  "number lines",
            input: "alpha\nbeta\n",
            fn:    NumberLines(),
            want:  "   1 | alpha\n   2 | beta\n",
        },
        {
            name:  "chain upper then number",
            input: "hello\n",
            fn:    Chain(Upper, NumberLines()),
            want:  "   1 | HELLO\n",
        },
        {
            name:  "empty input",
            input: "",
            fn:    Upper,
            want:  "",
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            r := strings.NewReader(tt.input)
            var buf bytes.Buffer
            err := Process(r, &buf, tt.fn)
            if err != nil {
                t.Fatalf("unexpected error: %v", err)
            }
            got := buf.String()
            if got != tt.want {
                t.Errorf("got %q, want %q", got, tt.want)
            }
        })
    }
}

func BenchmarkProcess(b *testing.B) {
    input := strings.Repeat("the quick brown fox\n", 1000)
    fn := Chain(Upper, NumberLines())

    for i := 0; i < b.N; i++ {
        r := strings.NewReader(input)
        var buf bytes.Buffer
        _ = Process(r, &buf, fn)
    }
}
```

### 3. New CLI: `cmd/hello-transform/main.go`

A Unix-style filter that reads from stdin or a file and writes to stdout or
a file:

```go
package main

import (
    "flag"
    "fmt"
    "log"
    "os"

    "github.com/itprodirect/go-hello-world/internal/transform"
)

func main() {
    mode := flag.String("mode", "upper", "transform mode: upper, lower, number, chain")
    inFile := flag.String("in", "", "input file (default: stdin)")
    outFile := flag.String("out", "", "output file (default: stdout)")
    flag.Parse()

    // Pick the transform function
    var fn transform.TransformFunc
    switch *mode {
    case "upper":
        fn = transform.Upper
    case "lower":
        fn = transform.Lower
    case "number":
        fn = transform.NumberLines()
    case "chain":
        fn = transform.Chain(transform.Upper, transform.NumberLines())
    default:
        log.Fatalf("unknown mode: %q", *mode)
    }

    // Open reader (file or stdin)
    var r *os.File
    if *inFile != "" {
        f, err := os.Open(*inFile)
        if err != nil {
            log.Fatalf("open input: %v", err)
        }
        defer f.Close()
        r = f
    } else {
        r = os.Stdin
    }

    // Open writer (file or stdout)
    var w *os.File
    if *outFile != "" {
        f, err := os.Create(*outFile)
        if err != nil {
            log.Fatalf("create output: %v", err)
        }
        defer f.Close()
        w = f
    } else {
        w = os.Stdout
    }

    if err := transform.Process(r, w, fn); err != nil {
        log.Fatalf("transform: %v", err)
    }

    if *outFile != "" {
        fmt.Fprintf(os.Stderr, "wrote output to %s\n", *outFile)
    }
}
```

### 4. Update Makefile

Add build and run targets:

```makefile
build:
	mkdir -p bin
	go build -o bin/hello-cli ./cmd/hello-cli
	go build -o bin/hello-server ./cmd/hello-server
	go build -o bin/hello-transform ./cmd/hello-transform

run-transform:
	echo "hello world\nfoo bar\nbaz" | go run ./cmd/hello-transform --mode chain
```

### 5. Update README

Add a section explaining the transform tool and the io.Reader/Writer pattern.

## Concepts Demonstrated

| Concept | Python/JS Equivalent |
|---------|---------------------|
| `io.Reader` interface | File-like objects / `Readable` streams |
| `io.Writer` interface | File-like objects / `Writable` streams |
| `bufio.Scanner` | `for line in file` / `readline` |
| `strings.NewReader` | `io.StringIO` / `Readable.from(string)` |
| `bytes.Buffer` | `io.BytesIO` / in-memory buffer |
| `os.Stdin` / `os.Stdout` | `sys.stdin` / `process.stdin` |
| Closures as `TransformFunc` | Lambda / arrow functions |
| `Chain()` composition | Function composition / pipe |

## Key Insight for Python/JS Devs

The `Process` function doesn't know or care whether it's reading from a file,
a network socket, a string, or a gzip stream. That's the power of coding to
interfaces instead of concrete types. In Python you'd use `typing.IO[str]`.
In Go it's just `io.Reader` — and every I/O thing in the standard library
already implements it.

## Verification

```bash
go test ./internal/transform/...
echo -e "hello\nworld" | go run ./cmd/hello-transform --mode upper
echo -e "hello\nworld" | go run ./cmd/hello-transform --mode number
go run ./cmd/hello-transform --mode chain --in go.mod
```
