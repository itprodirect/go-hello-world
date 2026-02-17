package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"strings"

	"github.com/itprodirect/go-hello-world/internal/apperror"
	"github.com/itprodirect/go-hello-world/internal/greeter"
	"github.com/itprodirect/go-hello-world/internal/metrics"
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
	name := flag.String("name", "world", "name to greet")
	repeat := flag.Int("repeat", 1, "number of greetings to generate")
	style := flag.String("style", "standard", "greeting style: standard, formal, shout")
	jsonOutput := flag.Bool("json", false, "emit JSON lines output")
	flag.Parse()

	if err := validateName(*name); err != nil {
		log.Fatalf("invalid input: %v", err)
	}
	if err := validateRepeat(*repeat); err != nil {
		log.Fatalf("invalid input: %v", err)
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
			payload := jsonGreeting{
				Index:   i + 1,
				Message: message,
			}

			line, err := json.Marshal(payload)
			if err != nil {
				log.Fatalf("marshal output: %v", err)
			}

			fmt.Println(string(line))
			continue
		}

		fmt.Println(message)
	}
}

func validateName(name string) error {
	clean := strings.TrimSpace(name)
	if clean == "" {
		return nil
	}
	for _, ch := range clean {
		if ch == '<' || ch == '>' || ch == '&' {
			return apperror.NewFieldError("name", "contains unsafe characters", apperror.ErrValidation)
		}
	}
	return nil
}

func validateRepeat(repeat int) error {
	if repeat < 1 || repeat > 1000 {
		return apperror.NewFieldError("repeat", "must be 1-1000", apperror.ErrValidation)
	}
	return nil
}
