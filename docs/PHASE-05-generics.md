# Phase 5: Generics & Reusable Data Structures

## Why This Matters (Python/JS → Go)

In Python, lists and dicts just hold anything. In TypeScript, you write
`Array<T>` and `Map<K, V>`. Go added generics in 1.18 — you can now write
type-safe, reusable data structures without `interface{}` casts everywhere.

This phase adds a small `collections` package with practical generic
utilities, plus a `Result` type that brings Rust-style error handling
to Go (something Python/JS devs will find interesting).

## What to Build

### 1. New package: `internal/collections/collections.go`

```go
package collections

// Map applies fn to each element and returns a new slice.
// Python: [fn(x) for x in items]
// JS:     items.map(fn)
func Map[T any, U any](items []T, fn func(T) U) []U {
    result := make([]U, len(items))
    for i, item := range items {
        result[i] = fn(item)
    }
    return result
}

// Filter returns elements where fn returns true.
// Python: [x for x in items if fn(x)]
// JS:     items.filter(fn)
func Filter[T any](items []T, fn func(T) bool) []T {
    var result []T
    for _, item := range items {
        if fn(item) {
            result = append(result, item)
        }
    }
    return result
}

// Reduce folds a slice into a single value.
// Python: functools.reduce(fn, items, initial)
// JS:     items.reduce(fn, initial)
func Reduce[T any, U any](items []T, initial U, fn func(U, T) U) U {
    acc := initial
    for _, item := range items {
        acc = fn(acc, item)
    }
    return acc
}

// Contains checks if a slice contains a value (requires comparable).
// Python: x in items
// JS:     items.includes(x)
func Contains[T comparable](items []T, target T) bool {
    for _, item := range items {
        if item == target {
            return true
        }
    }
    return false
}

// Unique returns a slice with duplicates removed, preserving order.
// Python: list(dict.fromkeys(items))
func Unique[T comparable](items []T) []T {
    seen := make(map[T]struct{}, len(items))
    var result []T
    for _, item := range items {
        if _, exists := seen[item]; !exists {
            seen[item] = struct{}{}
            result = append(result, item)
        }
    }
    return result
}

// GroupBy groups elements by a key function.
// Python: itertools.groupby (sort of)
// JS:     Object.groupBy (stage 3 proposal)
func GroupBy[T any, K comparable](items []T, keyFn func(T) K) map[K][]T {
    groups := make(map[K][]T)
    for _, item := range items {
        key := keyFn(item)
        groups[key] = append(groups[key], item)
    }
    return groups
}
```

### 2. New file: `internal/collections/result.go`

A generic `Result` type inspired by Rust — useful pattern for chaining:

```go
package collections

import "fmt"

// Result holds either a value or an error, never both.
// Inspired by Rust's Result<T, E>.
type Result[T any] struct {
    value T
    err   error
    ok    bool
}

// Ok creates a successful Result.
func Ok[T any](value T) Result[T] {
    return Result[T]{value: value, ok: true}
}

// Err creates a failed Result.
func Err[T any](err error) Result[T] {
    return Result[T]{err: err, ok: false}
}

// Unwrap returns the value or panics. Use only when you know it's Ok.
func (r Result[T]) Unwrap() T {
    if !r.ok {
        panic(fmt.Sprintf("called Unwrap on Err: %v", r.err))
    }
    return r.value
}

// UnwrapOr returns the value or the provided fallback.
func (r Result[T]) UnwrapOr(fallback T) T {
    if !r.ok {
        return fallback
    }
    return r.value
}

// IsOk returns true if the Result holds a value.
func (r Result[T]) IsOk() bool {
    return r.ok
}

// Error returns the error, or nil if Ok.
func (r Result[T]) Error() error {
    return r.err
}
```

### 3. Tests: `internal/collections/collections_test.go`

```go
package collections

import (
    "errors"
    "fmt"
    "strings"
    "testing"
)

func TestMap(t *testing.T) {
    nums := []int{1, 2, 3}
    doubled := Map(nums, func(n int) int { return n * 2 })

    want := []int{2, 4, 6}
    for i, v := range doubled {
        if v != want[i] {
            t.Errorf("Map[%d] = %d, want %d", i, v, want[i])
        }
    }
}

func TestMapTypeConversion(t *testing.T) {
    nums := []int{1, 2, 3}
    strs := Map(nums, func(n int) string {
        return fmt.Sprintf("#%d", n)
    })

    want := []string{"#1", "#2", "#3"}
    for i, v := range strs {
        if v != want[i] {
            t.Errorf("Map[%d] = %q, want %q", i, v, want[i])
        }
    }
}

func TestFilter(t *testing.T) {
    nums := []int{1, 2, 3, 4, 5, 6}
    evens := Filter(nums, func(n int) bool { return n%2 == 0 })

    want := []int{2, 4, 6}
    if len(evens) != len(want) {
        t.Fatalf("Filter length = %d, want %d", len(evens), len(want))
    }
    for i, v := range evens {
        if v != want[i] {
            t.Errorf("Filter[%d] = %d, want %d", i, v, want[i])
        }
    }
}

func TestReduce(t *testing.T) {
    nums := []int{1, 2, 3, 4}
    sum := Reduce(nums, 0, func(acc, n int) int { return acc + n })
    if sum != 10 {
        t.Errorf("Reduce sum = %d, want 10", sum)
    }
}

func TestReduceStringConcat(t *testing.T) {
    words := []string{"Go", "is", "fun"}
    sentence := Reduce(words, "", func(acc, w string) string {
        if acc == "" {
            return w
        }
        return acc + " " + w
    })
    if sentence != "Go is fun" {
        t.Errorf("Reduce concat = %q, want %q", sentence, "Go is fun")
    }
}

func TestContains(t *testing.T) {
    items := []string{"go", "python", "rust"}
    if !Contains(items, "go") {
        t.Error("expected Contains to find 'go'")
    }
    if Contains(items, "java") {
        t.Error("expected Contains to not find 'java'")
    }
}

func TestUnique(t *testing.T) {
    items := []int{1, 2, 2, 3, 1, 4}
    got := Unique(items)
    want := []int{1, 2, 3, 4}
    if len(got) != len(want) {
        t.Fatalf("Unique length = %d, want %d", len(got), len(want))
    }
    for i, v := range got {
        if v != want[i] {
            t.Errorf("Unique[%d] = %d, want %d", i, v, want[i])
        }
    }
}

func TestGroupBy(t *testing.T) {
    words := []string{"go", "great", "python", "power", "go"}
    groups := GroupBy(words, func(w string) string {
        return strings.ToUpper(w[:1])
    })

    if len(groups["G"]) != 3 {
        t.Errorf("G group length = %d, want 3", len(groups["G"]))
    }
    if len(groups["P"]) != 2 {
        t.Errorf("P group length = %d, want 2", len(groups["P"]))
    }
}

func TestResult_Ok(t *testing.T) {
    r := Ok(42)
    if !r.IsOk() {
        t.Fatal("expected Ok")
    }
    if r.Unwrap() != 42 {
        t.Fatalf("Unwrap = %d, want 42", r.Unwrap())
    }
}

func TestResult_Err(t *testing.T) {
    r := Err[int](errors.New("boom"))
    if r.IsOk() {
        t.Fatal("expected Err")
    }
    if r.UnwrapOr(99) != 99 {
        t.Fatalf("UnwrapOr = %d, want 99", r.UnwrapOr(99))
    }
}

func BenchmarkMap(b *testing.B) {
    nums := make([]int, 1000)
    for i := range nums {
        nums[i] = i
    }
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        Map(nums, func(n int) int { return n * 2 })
    }
}
```

### 4. Integration Example

Show how collections work with the greeter in a practical way. Add a
comment block or example in the README:

```go
// Generate greetings for a list of names:
names := []string{"Alice", "Bob", "Charlie", "Alice"}
unique := collections.Unique(names)
greetings := collections.Map(unique, func(name string) string {
    return greeter.BuildGreeting(name, 0)
})
// ["Hello, Alice!", "Hello, Bob!", "Hello, Charlie!"]
```

### 5. Update go.mod

Ensure `go 1.22` (or 1.18+) is set — generics require it. The current
`go 1.22` is already fine.

## Concepts Demonstrated

| Concept | Python/JS Equivalent |
|---------|---------------------|
| `[T any]` type parameter | `TypeVar('T')` / `<T>` in TS |
| `[T comparable]` constraint | Types that support `==` |
| Generic functions | `list(map(fn, items))` / `arr.map(fn)` |
| `struct{}` as set value | `set()` in Python |
| `Result[T]` | Rust's `Result<T, E>` / optional chaining |
| Zero-value generics | Default values by type |

## Key Insight for Python/JS Devs

Go generics are intentionally simpler than TypeScript's type system. There
are no conditional types, no mapped types, no `infer`. You get type
parameters with constraints and that's it. This is by design — Go values
simplicity and readability over type-level expressiveness.

The `comparable` constraint is the most useful built-in: it means "any type
that supports `==`". For numeric constraints, see `golang.org/x/exp/constraints`.

## Verification

```bash
go test ./internal/collections/...
make bench  # see generic function performance
make test   # everything still passes
```
