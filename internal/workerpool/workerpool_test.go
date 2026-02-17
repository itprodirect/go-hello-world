package workerpool

import (
	"context"
	"fmt"
	"sort"
	"sync/atomic"
	"testing"
	"time"
)

func TestPoolRunBasic(t *testing.T) {
	pool := New[int, string](4)
	inputs := []int{1, 2, 3, 4, 5}

	results := pool.Run(context.Background(), inputs, func(ctx context.Context, n int) string {
		return fmt.Sprintf("item_%d", n)
	})

	if len(results) != len(inputs) {
		t.Fatalf("got %d results, want %d", len(results), len(inputs))
	}

	sort.Strings(results)
	want := []string{"item_1", "item_2", "item_3", "item_4", "item_5"}
	for i := range want {
		if results[i] != want[i] {
			t.Errorf("results[%d] = %q, want %q", i, results[i], want[i])
		}
	}
}

func TestPoolRunEmptyInput(t *testing.T) {
	pool := New[int, int](2)
	results := pool.Run(context.Background(), nil, func(ctx context.Context, n int) int {
		return n
	})

	if results != nil {
		t.Fatalf("expected nil, got %v", results)
	}
}

func TestPoolRunConcurrency(t *testing.T) {
	pool := New[int, int](4)
	inputs := make([]int, 20)
	for i := range inputs {
		inputs[i] = i
	}

	var maxConcurrent atomic.Int32
	var current atomic.Int32

	results := pool.Run(context.Background(), inputs, func(ctx context.Context, n int) int {
		c := current.Add(1)
		for {
			old := maxConcurrent.Load()
			if c <= old || maxConcurrent.CompareAndSwap(old, c) {
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
		current.Add(-1)
		return n * 2
	})

	if len(results) != len(inputs) {
		t.Fatalf("got %d results, want %d", len(results), len(inputs))
	}

	if maxConcurrent.Load() < 2 {
		t.Errorf("expected concurrent execution, max concurrent=%d", maxConcurrent.Load())
	}
}

func TestPoolRunContextCancellation(t *testing.T) {
	pool := New[int, int](2)
	inputs := make([]int, 100)
	for i := range inputs {
		inputs[i] = i
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	results := pool.Run(ctx, inputs, func(ctx context.Context, n int) int {
		time.Sleep(20 * time.Millisecond)
		return n
	})

	if len(results) >= len(inputs) {
		t.Fatalf("expected early stop on cancellation, got %d", len(results))
	}
}

func BenchmarkPoolRun(b *testing.B) {
	pool := New[int, int](8)
	inputs := make([]int, 1000)
	for i := range inputs {
		inputs[i] = i
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pool.Run(context.Background(), inputs, func(ctx context.Context, n int) int {
			return n * 2
		})
	}
}
