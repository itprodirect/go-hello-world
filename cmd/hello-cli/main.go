package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"

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
	jsonOutput := flag.Bool("json", false, "emit JSON lines output")
	flag.Parse()

	if *repeat < 1 {
		log.Fatal("--repeat must be >= 1")
	}

	counters := metrics.NewCounters()

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
				message := greeter.BuildGreeting(*name, sequence)
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
