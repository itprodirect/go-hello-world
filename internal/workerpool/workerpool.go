package workerpool

import (
	"context"
	"sync"
)

// TaskFunc processes one input item and returns one output item.
type TaskFunc[In any, Out any] func(ctx context.Context, input In) Out

// Pool runs tasks with a fixed worker count.
type Pool[In any, Out any] struct {
	workers int
}

// New returns a pool with at least one worker.
func New[In any, Out any](workers int) *Pool[In, Out] {
	if workers < 1 {
		workers = 1
	}
	return &Pool[In, Out]{workers: workers}
}

// Run fans out inputs and collects outputs in arbitrary order.
func (p *Pool[In, Out]) Run(ctx context.Context, inputs []In, fn TaskFunc[In, Out]) []Out {
	if len(inputs) == 0 {
		return nil
	}

	jobs := make(chan In)
	results := make(chan Out, len(inputs))

	var wg sync.WaitGroup
	for i := 0; i < p.workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for input := range jobs {
				select {
				case <-ctx.Done():
					return
				default:
				}

				out := fn(ctx, input)
				select {
				case <-ctx.Done():
					return
				case results <- out:
				}
			}
		}()
	}

	go func() {
		defer close(jobs)
		for _, input := range inputs {
			select {
			case <-ctx.Done():
				return
			case jobs <- input:
			}
		}
	}()

	go func() {
		wg.Wait()
		close(results)
	}()

	out := make([]Out, 0, len(inputs))
	for result := range results {
		out = append(out, result)
	}

	return out
}
