package pipeline

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"sync"
)

const maxScanTokenSize = 1024 * 1024

// Stage transforms a line of text. Return empty string to drop the line.
type Stage func(line string) string

// Run reads lines from r, pushes them through stages in order,
// and writes surviving lines to w. Returns count of lines written.
func Run(ctx context.Context, r io.Reader, w io.Writer, stages ...Stage) (int, error) {
	scanner := newScanner(r)
	written := 0

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return written, ctx.Err()
		default:
		}

		line := applyStages(scanner.Text(), stages)
		if line == "" {
			continue
		}

		if _, err := fmt.Fprintln(w, line); err != nil {
			return written, fmt.Errorf("write: %w", err)
		}
		written++
	}

	if err := scanner.Err(); err != nil {
		return written, fmt.Errorf("scan: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return written, err
	}
	return written, nil
}

// RunConcurrent fans out lines across multiple worker goroutines.
// Output order is not preserved.
func RunConcurrent(ctx context.Context, r io.Reader, w io.Writer, workers int, stages ...Stage) (int, error) {
	if workers < 1 {
		workers = 1
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	lines := make(chan string, workers*2)
	results := make(chan string, workers*2)

	var workerWG sync.WaitGroup
	for i := 0; i < workers; i++ {
		workerWG.Add(1)
		go func() {
			defer workerWG.Done()
			runWorker(ctx, lines, results, stages)
		}()
	}

	readErrCh := make(chan error, 1)
	go func() {
		readErrCh <- feedLines(ctx, r, lines)
	}()

	go func() {
		workerWG.Wait()
		close(results)
	}()

	written := 0
	var writeErr error
	for line := range results {
		if writeErr != nil {
			continue
		}

		if _, err := fmt.Fprintln(w, line); err != nil {
			writeErr = fmt.Errorf("write: %w", err)
			cancel()
			continue
		}
		written++
	}

	readErr := <-readErrCh

	if writeErr != nil {
		return written, writeErr
	}

	if readErr != nil && !errors.Is(readErr, context.Canceled) && !errors.Is(readErr, context.DeadlineExceeded) {
		return written, fmt.Errorf("scan: %w", readErr)
	}

	if err := ctx.Err(); err != nil {
		return written, err
	}
	return written, nil
}

// Chain composes multiple stages into one.
func Chain(stages ...Stage) Stage {
	return func(line string) string {
		return applyStages(line, stages)
	}
}

func runWorker(ctx context.Context, lines <-chan string, results chan<- string, stages []Stage) {
	for {
		select {
		case <-ctx.Done():
			return
		case line, ok := <-lines:
			if !ok {
				return
			}

			out := applyStages(line, stages)
			if out == "" {
				continue
			}

			select {
			case <-ctx.Done():
				return
			case results <- out:
			}
		}
	}
}

func feedLines(ctx context.Context, r io.Reader, lines chan<- string) error {
	defer close(lines)

	scanner := newScanner(r)
	for scanner.Scan() {
		line := scanner.Text()

		select {
		case <-ctx.Done():
			return ctx.Err()
		case lines <- line:
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return nil
}

func applyStages(line string, stages []Stage) string {
	out := line
	for _, stage := range stages {
		out = stage(out)
		if out == "" {
			return ""
		}
	}
	return out
}

func newScanner(r io.Reader) *bufio.Scanner {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), maxScanTokenSize)
	return scanner
}
