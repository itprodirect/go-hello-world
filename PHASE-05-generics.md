# Phase 5: Generics Toolkit + TTL Cache

> **Produces:** `internal/collections`, `internal/cache`
> **Teaches:** Type parameters, constraints, `comparable`, generic data structures
> **Ships:** Reusable utility library for every future Go project

## Implementation Status (February 17, 2026)

- Status: Pending
- Planned after Phase 4 is complete.
- Dependency status: foundational phases are complete and verified.

## Why Generics Matter for Your Stack

Before Go 1.18, you'd write `Map`, `Filter`, `Reduce` for `[]int`, then
again for `[]string`, then again for `[]MyStruct`. Generics fix that —
write once, use with any type, fully type-checked at compile time.

The `collections` package gives you Python/JS array methods in Go.
The `cache` package gives you a production-ready in-memory TTL cache
that's useful for health check result caching, API response caching,
and rate limiting state.

---

## Package 1: `internal/collections/collections.go`

Functional utilities for slices — the Go equivalent of Python list
comprehensions and JS array methods.

```go
package collections

import "fmt"

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

// Reduce folds a slice into a single value left-to-right.
// Python: functools.reduce(fn, items, initial)
// JS:     items.reduce(fn, initial)
func Reduce[T any, U any](items []T, initial U, fn func(U, T) U) U {
	acc := initial
	for _, item := range items {
		acc = fn(acc, item)
	}
	return acc
}

// Contains checks membership.
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

// Unique removes duplicates while preserving order.
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
// Python: roughly itertools.groupby (but works on unsorted input)
func GroupBy[T any, K comparable](items []T, keyFn func(T) K) map[K][]T {
	groups := make(map[K][]T)
	for _, item := range items {
		key := keyFn(item)
		groups[key] = append(groups[key], item)
	}
	return groups
}

// Find returns the first element matching fn, or zero value + false.
// Python: next((x for x in items if fn(x)), None)
// JS:     items.find(fn)
func Find[T any](items []T, fn func(T) bool) (T, bool) {
	for _, item := range items {
		if fn(item) {
			return item, true
		}
	}
	var zero T
	return zero, false
}

// Count returns how many elements match fn.
// Python: sum(1 for x in items if fn(x))
func Count[T any](items []T, fn func(T) bool) int {
	n := 0
	for _, item := range items {
		if fn(item) {
			n++
		}
	}
	return n
}

// Chunk splits a slice into groups of size n.
// Python: [items[i:i+n] for i in range(0, len(items), n)]
func Chunk[T any](items []T, size int) [][]T {
	if size <= 0 {
		return nil
	}
	var chunks [][]T
	for i := 0; i < len(items); i += size {
		end := i + size
		if end > len(items) {
			end = len(items)
		}
		chunks = append(chunks, items[i:end])
	}
	return chunks
}

// ToMap converts a slice to a map using key and value functions.
// Python: {key(x): val(x) for x in items}
func ToMap[T any, K comparable, V any](items []T, keyFn func(T) K, valFn func(T) V) map[K]V {
	m := make(map[K]V, len(items))
	for _, item := range items {
		m[keyFn(item)] = valFn(item)
	}
	return m
}

// Flatten concatenates nested slices into one.
// Python: [item for sublist in items for item in sublist]
// JS:     items.flat()
func Flatten[T any](items [][]T) []T {
	var result []T
	for _, sub := range items {
		result = append(result, sub...)
	}
	return result
}

// ForEach runs fn on each element (for side effects).
func ForEach[T any](items []T, fn func(int, T)) {
	for i, item := range items {
		fn(i, item)
	}
}

// Stringer converts any slice to string representations.
func Stringer[T any](items []T) []string {
	return Map(items, func(item T) string {
		return fmt.Sprintf("%v", item)
	})
}
```

### `internal/collections/result.go`

A generic `Result` type inspired by Rust — useful for explicit error handling
without panics. Pairs well with `Map` and other collection functions.

```go
package collections

import "fmt"

// Result holds either a value or an error.
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

// IsOk returns true if the Result holds a value.
func (r Result[T]) IsOk() bool {
	return r.ok
}

// Unwrap returns the value or panics.
func (r Result[T]) Unwrap() T {
	if !r.ok {
		panic(fmt.Sprintf("Unwrap on Err: %v", r.err))
	}
	return r.value
}

// UnwrapOr returns the value or the fallback.
func (r Result[T]) UnwrapOr(fallback T) T {
	if !r.ok {
		return fallback
	}
	return r.value
}

// UnwrapErr returns the error (nil if Ok).
func (r Result[T]) UnwrapErr() error {
	return r.err
}

// MapResult transforms the value inside a Result if Ok.
func MapResult[T any, U any](r Result[T], fn func(T) U) Result[U] {
	if !r.ok {
		return Err[U](r.err)
	}
	return Ok(fn(r.value))
}
```

---

### Test: `internal/collections/collections_test.go`

```go
package collections

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestMap(t *testing.T) {
	got := Map([]int{1, 2, 3}, func(n int) int { return n * 2 })
	want := []int{2, 4, 6}
	assertSliceEqual(t, got, want)
}

func TestMap_TypeConversion(t *testing.T) {
	got := Map([]int{1, 2, 3}, func(n int) string {
		return fmt.Sprintf("#%d", n)
	})
	want := []string{"#1", "#2", "#3"}
	assertSliceEqual(t, got, want)
}

func TestFilter(t *testing.T) {
	got := Filter([]int{1, 2, 3, 4, 5, 6}, func(n int) bool { return n%2 == 0 })
	want := []int{2, 4, 6}
	assertSliceEqual(t, got, want)
}

func TestReduce(t *testing.T) {
	sum := Reduce([]int{1, 2, 3, 4}, 0, func(acc, n int) int { return acc + n })
	if sum != 10 {
		t.Errorf("Reduce sum = %d, want 10", sum)
	}
}

func TestReduceStringConcat(t *testing.T) {
	got := Reduce([]string{"Go", "is", "great"}, "", func(acc, w string) string {
		if acc == "" {
			return w
		}
		return acc + " " + w
	})
	if got != "Go is great" {
		t.Errorf("Reduce = %q", got)
	}
}

func TestContains(t *testing.T) {
	items := []string{"go", "python", "rust"}
	if !Contains(items, "go") {
		t.Error("should contain go")
	}
	if Contains(items, "java") {
		t.Error("should not contain java")
	}
}

func TestUnique(t *testing.T) {
	got := Unique([]int{1, 2, 2, 3, 1, 4})
	want := []int{1, 2, 3, 4}
	assertSliceEqual(t, got, want)
}

func TestGroupBy(t *testing.T) {
	words := []string{"go", "great", "python", "power"}
	groups := GroupBy(words, func(w string) string {
		return strings.ToUpper(w[:1])
	})
	if len(groups["G"]) != 2 {
		t.Errorf("G group = %d, want 2", len(groups["G"]))
	}
	if len(groups["P"]) != 2 {
		t.Errorf("P group = %d, want 2", len(groups["P"]))
	}
}

func TestFind(t *testing.T) {
	items := []int{1, 2, 3, 4, 5}
	val, ok := Find(items, func(n int) bool { return n > 3 })
	if !ok || val != 4 {
		t.Errorf("Find = %d, %v; want 4, true", val, ok)
	}

	_, ok = Find(items, func(n int) bool { return n > 10 })
	if ok {
		t.Error("Find should return false for no match")
	}
}

func TestCount(t *testing.T) {
	items := []int{1, 2, 3, 4, 5, 6}
	got := Count(items, func(n int) bool { return n%2 == 0 })
	if got != 3 {
		t.Errorf("Count = %d, want 3", got)
	}
}

func TestChunk(t *testing.T) {
	got := Chunk([]int{1, 2, 3, 4, 5}, 2)
	if len(got) != 3 {
		t.Fatalf("Chunk count = %d, want 3", len(got))
	}
	assertSliceEqual(t, got[0], []int{1, 2})
	assertSliceEqual(t, got[1], []int{3, 4})
	assertSliceEqual(t, got[2], []int{5})
}

func TestFlatten(t *testing.T) {
	got := Flatten([][]int{{1, 2}, {3}, {4, 5}})
	want := []int{1, 2, 3, 4, 5}
	assertSliceEqual(t, got, want)
}

func TestToMap(t *testing.T) {
	type user struct {
		ID   int
		Name string
	}
	users := []user{{1, "Nick"}, {2, "Alice"}}
	m := ToMap(users,
		func(u user) int { return u.ID },
		func(u user) string { return u.Name },
	)
	if m[1] != "Nick" || m[2] != "Alice" {
		t.Errorf("ToMap = %v", m)
	}
}

// --- Result tests ---

func TestResult_Ok(t *testing.T) {
	r := Ok(42)
	if !r.IsOk() {
		t.Fatal("expected Ok")
	}
	if r.Unwrap() != 42 {
		t.Errorf("Unwrap = %d, want 42", r.Unwrap())
	}
}

func TestResult_Err(t *testing.T) {
	r := Err[int](errors.New("boom"))
	if r.IsOk() {
		t.Fatal("expected Err")
	}
	if r.UnwrapOr(99) != 99 {
		t.Errorf("UnwrapOr = %d, want 99", r.UnwrapOr(99))
	}
	if r.UnwrapErr() == nil {
		t.Error("UnwrapErr should return error")
	}
}

func TestMapResult(t *testing.T) {
	r := Ok(5)
	doubled := MapResult(r, func(n int) int { return n * 2 })
	if doubled.Unwrap() != 10 {
		t.Errorf("MapResult = %d, want 10", doubled.Unwrap())
	}

	e := Err[int](errors.New("fail"))
	mapped := MapResult(e, func(n int) int { return n * 2 })
	if mapped.IsOk() {
		t.Error("MapResult on Err should remain Err")
	}
}

// --- Benchmarks ---

func BenchmarkMap(b *testing.B) {
	nums := make([]int, 10000)
	for i := range nums {
		nums[i] = i
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Map(nums, func(n int) int { return n * 2 })
	}
}

func BenchmarkFilter(b *testing.B) {
	nums := make([]int, 10000)
	for i := range nums {
		nums[i] = i
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Filter(nums, func(n int) bool { return n%2 == 0 })
	}
}

func BenchmarkUnique(b *testing.B) {
	items := make([]int, 10000)
	for i := range items {
		items[i] = i % 100 // lots of dupes
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Unique(items)
	}
}

// --- Helpers ---

func assertSliceEqual[T comparable](t *testing.T, got, want []T) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("length = %d, want %d; got %v", len(got), len(want), got)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("[%d] = %v, want %v", i, got[i], want[i])
		}
	}
}
```

---

## Package 2: `internal/cache/cache.go`

A generic, thread-safe, TTL-based in-memory cache. Use it to cache health
check results, API responses, or anything that's expensive to compute.

```go
package cache

import (
	"sync"
	"time"
)

// entry stores a cached value with its expiration time.
type entry[V any] struct {
	value     V
	expiresAt time.Time
}

// Cache is a thread-safe generic TTL cache.
// Python equivalent: cachetools.TTLCache
type Cache[K comparable, V any] struct {
	mu      sync.RWMutex
	items   map[K]entry[V]
	ttl     time.Duration
	nowFunc func() time.Time // injectable clock for testing
}

// New creates a cache with the given default TTL.
func New[K comparable, V any](ttl time.Duration) *Cache[K, V] {
	return &Cache[K, V]{
		items:   make(map[K]entry[V]),
		ttl:     ttl,
		nowFunc: time.Now,
	}
}

// Get retrieves a value. Returns (value, true) if found and not expired.
func (c *Cache[K, V]) Get(key K) (V, bool) {
	c.mu.RLock()
	e, exists := c.items[key]
	c.mu.RUnlock()

	if !exists || c.nowFunc().After(e.expiresAt) {
		var zero V
		return zero, false
	}
	return e.value, true
}

// Set stores a value with the default TTL.
func (c *Cache[K, V]) Set(key K, value V) {
	c.SetWithTTL(key, value, c.ttl)
}

// SetWithTTL stores a value with a custom TTL.
func (c *Cache[K, V]) SetWithTTL(key K, value V, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = entry[V]{
		value:     value,
		expiresAt: c.nowFunc().Add(ttl),
	}
}

// Delete removes a key.
func (c *Cache[K, V]) Delete(key K) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.items, key)
}

// Len returns the number of items (including expired, pre-cleanup).
func (c *Cache[K, V]) Len() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.items)
}

// Cleanup removes all expired entries. Call periodically or before Len().
func (c *Cache[K, V]) Cleanup() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := c.nowFunc()
	removed := 0
	for key, e := range c.items {
		if now.After(e.expiresAt) {
			delete(c.items, key)
			removed++
		}
	}
	return removed
}

// GetOrSet returns the cached value if present, otherwise calls fn,
// caches the result, and returns it. This is the most useful method.
// Python: cache.setdefault() but with a factory function.
func (c *Cache[K, V]) GetOrSet(key K, fn func() V) V {
	if val, ok := c.Get(key); ok {
		return val
	}

	val := fn()
	c.Set(key, val)
	return val
}

// Keys returns all non-expired keys.
func (c *Cache[K, V]) Keys() []K {
	c.mu.RLock()
	defer c.mu.RUnlock()

	now := c.nowFunc()
	var keys []K
	for k, e := range c.items {
		if now.Before(e.expiresAt) {
			keys = append(keys, k)
		}
	}
	return keys
}
```

### Test: `internal/cache/cache_test.go`

```go
package cache

import (
	"sync"
	"testing"
	"time"
)

func TestCache_SetAndGet(t *testing.T) {
	c := New[string, int](5 * time.Minute)
	c.Set("count", 42)

	val, ok := c.Get("count")
	if !ok || val != 42 {
		t.Fatalf("Get = (%d, %v), want (42, true)", val, ok)
	}
}

func TestCache_MissReturnsZero(t *testing.T) {
	c := New[string, string](time.Minute)

	val, ok := c.Get("missing")
	if ok || val != "" {
		t.Fatalf("Get = (%q, %v), want (\"\", false)", val, ok)
	}
}

func TestCache_TTLExpiration(t *testing.T) {
	// Use a fake clock so we don't need to sleep
	now := time.Now()
	c := New[string, int](100 * time.Millisecond)
	c.nowFunc = func() time.Time { return now }

	c.Set("key", 1)

	// Before expiry
	val, ok := c.Get("key")
	if !ok || val != 1 {
		t.Fatalf("before expiry: Get = (%d, %v)", val, ok)
	}

	// Advance time past TTL
	now = now.Add(200 * time.Millisecond)

	_, ok = c.Get("key")
	if ok {
		t.Fatal("after expiry: Get should return false")
	}
}

func TestCache_Delete(t *testing.T) {
	c := New[string, int](time.Minute)
	c.Set("key", 1)
	c.Delete("key")

	_, ok := c.Get("key")
	if ok {
		t.Fatal("Get after Delete should return false")
	}
}

func TestCache_Cleanup(t *testing.T) {
	now := time.Now()
	c := New[string, int](100 * time.Millisecond)
	c.nowFunc = func() time.Time { return now }

	c.Set("a", 1)
	c.Set("b", 2)
	c.SetWithTTL("c", 3, time.Hour) // long TTL

	// Expire a and b
	now = now.Add(200 * time.Millisecond)

	removed := c.Cleanup()
	if removed != 2 {
		t.Errorf("Cleanup removed %d, want 2", removed)
	}
	if c.Len() != 1 {
		t.Errorf("Len = %d, want 1", c.Len())
	}
}

func TestCache_GetOrSet(t *testing.T) {
	c := New[string, int](time.Minute)
	calls := 0

	factory := func() int {
		calls++
		return 42
	}

	val1 := c.GetOrSet("key", factory)
	val2 := c.GetOrSet("key", factory)

	if val1 != 42 || val2 != 42 {
		t.Errorf("values = %d, %d; want 42, 42", val1, val2)
	}
	if calls != 1 {
		t.Errorf("factory called %d times, want 1", calls)
	}
}

func TestCache_Keys(t *testing.T) {
	c := New[string, int](time.Minute)
	c.Set("a", 1)
	c.Set("b", 2)

	keys := c.Keys()
	if len(keys) != 2 {
		t.Errorf("Keys count = %d, want 2", len(keys))
	}
}

func TestCache_ConcurrentAccess(t *testing.T) {
	c := New[int, int](time.Minute)
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			c.Set(n, n*2)
			c.Get(n)
		}(i)
	}

	wg.Wait()

	if c.Len() != 100 {
		t.Errorf("Len = %d, want 100", c.Len())
	}
}

func BenchmarkCache_SetGet(b *testing.B) {
	c := New[int, int](time.Minute)
	for i := 0; i < b.N; i++ {
		c.Set(i%1000, i)
		c.Get(i % 1000)
	}
}

func BenchmarkCache_ConcurrentSetGet(b *testing.B) {
	c := New[int, int](time.Minute)
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			c.Set(i%1000, i)
			c.Get(i % 1000)
			i++
		}
	})
}
```

---

## Real-World Composition Examples

### Cache health check results

```go
import (
    "github.com/itprodirect/go-hello-world/internal/cache"
    "github.com/itprodirect/go-hello-world/internal/checker"
)

resultCache := cache.New[string, checker.Result](2 * time.Minute)

// In the health check handler:
result := resultCache.GetOrSet(target.URL, func() checker.Result {
    return checker.Check(ctx, target)
})
```

### Filter and transform with collections

```go
import "github.com/itprodirect/go-hello-world/internal/collections"

results := pool.Run(ctx, targets, checker.Check)

// Get just the failures
failures := collections.Filter(results, func(r checker.Result) bool {
    return r.Status == "down"
})

// Extract just the names
names := collections.Map(failures, func(r checker.Result) string {
    return r.Name
})

// Group by status
byStatus := collections.GroupBy(results, func(r checker.Result) string {
    return r.Status
})
```

---

## Concepts Demonstrated

| Go Pattern | Python/JS Equivalent | Why It's Useful |
|-----------|---------------------|----------------|
| `[T any]` type param | `TypeVar('T')` / `<T>` | Write once, use with any type |
| `[T comparable]` constraint | Types that support `==` | Map keys, dedup, Contains |
| `Map[T, U]` (two type params) | `map(fn, items)` | Type-safe input→output conversion |
| `sync.RWMutex` in cache | `threading.RLock` | Many readers, one writer |
| Injectable `nowFunc` | Mocking `time.time()` | Test TTL without sleeping |
| `GetOrSet` pattern | `@lru_cache` / `setdefault` | Compute-once, cache forever |
| `Result[T]` | Rust's `Result<T, E>` | Explicit error handling without panics |

---

## Verification

```bash
go test ./internal/collections/...
go test ./internal/cache/...
go test ./...

make bench  # see generic function performance + cache throughput
```
