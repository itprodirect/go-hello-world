package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/itprodirect/go-hello-world/internal/checker"
	"github.com/itprodirect/go-hello-world/internal/workerpool"
)

type checkFunc func(context.Context, checker.Target) checker.Result

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	return runWithChecker(args, stdout, stderr, checker.Check)
}

func runWithChecker(args []string, stdout, stderr io.Writer, check checkFunc) int {
	fs := flag.NewFlagSet("healthcheck", flag.ContinueOnError)
	fs.SetOutput(stderr)

	targetsFile := fs.String("targets", "", "path to targets JSON file")
	workers := fs.Int("workers", 8, "number of concurrent workers")
	timeout := fs.Int("timeout", 5000, "default timeout per check in ms")
	jsonOutput := fs.Bool("json", false, "output results as JSON lines")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *workers < 1 {
		fmt.Fprintf(stderr, "invalid workers: %d (must be >= 1)\n", *workers)
		return 1
	}
	if *timeout < 1 {
		fmt.Fprintf(stderr, "invalid timeout: %d (must be >= 1ms)\n", *timeout)
		return 1
	}

	var targets []checker.Target
	if *targetsFile == "" {
		fmt.Fprintln(stderr, "No targets file provided. Using built-in demo targets.")
		targets = demoTargets()
	} else {
		loaded, err := checker.LoadTargets(*targetsFile)
		if err != nil {
			fmt.Fprintf(stderr, "load targets: %v\n", err)
			return 1
		}
		targets = loaded
	}

	for i := range targets {
		if targets[i].Timeout <= 0 {
			targets[i].Timeout = *timeout
		}
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	start := time.Now()
	pool := workerpool.New[checker.Target, checker.Result](*workers)
	results := pool.Run(ctx, targets, workerpool.TaskFunc[checker.Target, checker.Result](check))
	elapsed := time.Since(start)

	if *jsonOutput {
		encoder := json.NewEncoder(stdout)
		for _, result := range results {
			if err := encoder.Encode(result); err != nil {
				fmt.Fprintf(stderr, "encode result: %v\n", err)
				return 1
			}
		}
	} else {
		if err := printTable(stdout, results); err != nil {
			fmt.Fprintf(stderr, "render table: %v\n", err)
			return 1
		}
	}

	up, down, errCount := 0, 0, 0
	for _, result := range results {
		switch result.Status {
		case "up":
			up++
		case "down":
			down++
		default:
			errCount++
		}
	}

	fmt.Fprintf(
		stderr,
		"\n--- %d checks in %s | %d up | %d down | %d errors ---\n",
		len(results),
		elapsed.Round(time.Millisecond),
		up,
		down,
		errCount,
	)

	if down > 0 || errCount > 0 {
		return 1
	}

	return 0
}

func printTable(w io.Writer, results []checker.Result) error {
	writer := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(writer, "STATUS\tNAME\tTYPE\tTARGET\tLATENCY\tDETAIL")
	fmt.Fprintln(writer, "------\t----\t----\t------\t-------\t------")

	for _, result := range results {
		fmt.Fprintf(
			writer,
			"%s\t%s\t%s\t%s\t%s\t%s",
			checker.StatusEmoji(result.Status),
			result.Name,
			result.Type,
			result.Target,
			result.Latency.Round(time.Millisecond),
			result.Detail,
		)
		if result.TLS != nil {
			fmt.Fprintf(writer, " (TLS: %d days left)", result.TLS.DaysLeft)
		}
		fmt.Fprintln(writer)
	}

	return writer.Flush()
}

func demoTargets() []checker.Target {
	return []checker.Target{
		{Name: "google", URL: "https://www.google.com", Type: "http"},
		{Name: "github", URL: "https://github.com", Type: "http"},
		{Name: "localhost-8080", Host: "localhost", Port: 8080, Type: "tcp"},
		{Name: "dns-google", Host: "dns.google", Type: "dns"},
	}
}
