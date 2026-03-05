package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/itprodirect/go-hello-world/internal/greeter"
	"github.com/itprodirect/go-hello-world/internal/metrics"
	"github.com/itprodirect/go-hello-world/internal/validator"
)

type workerResult struct {
	index   int
	message string
}

type jsonGreeting struct {
	Index   int    `json:"index"`
	Message string `json:"message"`
}

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("hello-cli", flag.ContinueOnError)
	fs.SetOutput(stderr)

	name := fs.String("name", "world", "name to greet")
	repeat := fs.Int("repeat", 1, "number of greetings to generate")
	style := fs.String("style", "standard", "greeting style: standard, formal, shout")
	jsonOutput := fs.Bool("json", false, "emit JSON lines output")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if err := validator.ValidateName(*name); err != nil {
		fmt.Fprintf(stderr, "invalid input: %v\n", err)
		return 1
	}
	if err := validator.ValidateRepeat(*repeat); err != nil {
		fmt.Fprintf(stderr, "invalid input: %v\n", err)
		return 1
	}

	counters := metrics.NewCounters()
	g := greeter.New(*style)

	jobs := make(chan int)
	results := make(chan workerResult, *repeat)

	workerCount := *repeat
	if workerCount > 4 {
		workerCount = 4
	}

	for i := 0; i < workerCount; i++ {
		go func() {
			for idx := range jobs {
				sequence := idx + 1
				message := g.Greet(*name, sequence)
				counters.Inc("cli_greetings_generated")
				results <- workerResult{index: idx, message: message}
			}
		}()
	}

	for i := 0; i < *repeat; i++ {
		jobs <- i
	}
	close(jobs)

	orderedMessages := make([]string, *repeat)
	for i := 0; i < *repeat; i++ {
		result := <-results
		orderedMessages[result.index] = result.message
	}

	for i, message := range orderedMessages {
		if *jsonOutput {
			payload := jsonGreeting{Index: i + 1, Message: message}
			line, err := json.Marshal(payload)
			if err != nil {
				fmt.Fprintf(stderr, "marshal output: %v\n", err)
				return 1
			}
			fmt.Fprintln(stdout, string(line))
			continue
		}
		fmt.Fprintln(stdout, message)
	}

	return 0
}
