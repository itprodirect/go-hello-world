package metrics

import (
	"sync"
	"testing"
)

func TestCountersIncAndGet(t *testing.T) {
	counters := NewCounters()

	counters.Inc("hello_requests")
	counters.Add("hello_requests", 2)

	got := counters.Get("hello_requests")
	if got != 3 {
		t.Fatalf("Get() = %d, want 3", got)
	}
}

func TestCountersConcurrentIncrement(t *testing.T) {
	counters := NewCounters()

	const workers = 100
	var wg sync.WaitGroup
	wg.Add(workers)

	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			counters.Inc("shared_counter")
		}()
	}

	wg.Wait()

	got := counters.Get("shared_counter")
	if got != workers {
		t.Fatalf("Get() = %d, want %d", got, workers)
	}
}
