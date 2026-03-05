package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/itprodirect/go-hello-world/internal/pipeline"
	"github.com/itprodirect/go-hello-world/internal/transform"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	os.Exit(runWithContext(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	return runWithContext(context.Background(), args, stdin, stdout, stderr)
}

func runWithContext(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("dataflow", flag.ContinueOnError)
	fs.SetOutput(stderr)

	mode := fs.String("mode", "upper", "transform mode")
	match := fs.String("match", "", "substring for grep/filter or drop modes")
	field := fs.String("field", "", "JSON field name for json-field mode")
	oldStr := fs.String("old", "", "string to replace for replace mode")
	newStr := fs.String("new", "", "replacement string for replace mode")
	inFile := fs.String("in", "", "input file path (default: stdin)")
	outFile := fs.String("out", "", "output file path (default: stdout)")
	workers := fs.Int("workers", 0, "concurrent workers (0 = sequential)")

	if err := fs.Parse(args); err != nil {
		return 2
	}

	if *workers < 0 {
		fmt.Fprintf(stderr, "invalid workers: %d (must be >= 0)\n", *workers)
		return 1
	}

	stages, err := buildStages(*mode, *match, *field, *oldStr, *newStr)
	if err != nil {
		fmt.Fprintf(stderr, "invalid mode/options: %v\n", err)
		return 1
	}

	input := stdin
	var inputFile *os.File
	if *inFile != "" {
		f, err := os.Open(*inFile)
		if err != nil {
			fmt.Fprintf(stderr, "open input: %v\n", err)
			return 1
		}
		inputFile = f
		input = f
	}
	if inputFile != nil {
		defer inputFile.Close()
	}

	output := stdout
	var outputFile *os.File
	if *outFile != "" {
		f, err := os.Create(*outFile)
		if err != nil {
			fmt.Fprintf(stderr, "create output: %v\n", err)
			return 1
		}
		outputFile = f
		output = f
	}
	if outputFile != nil {
		defer outputFile.Close()
	}

	var written int
	if *workers > 0 {
		written, err = pipeline.RunConcurrent(ctx, input, output, *workers, stages...)
	} else {
		written, err = pipeline.Run(ctx, input, output, stages...)
	}
	if err != nil {
		fmt.Fprintf(stderr, "pipeline error: %v\n", err)
		return 1
	}

	if *outFile != "" {
		fmt.Fprintf(stderr, "wrote %d lines to %s\n", written, *outFile)
	}

	return 0
}

func buildStages(mode, match, field, oldStr, newStr string) ([]pipeline.Stage, error) {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "upper":
		return []pipeline.Stage{transform.Upper}, nil
	case "lower":
		return []pipeline.Stage{transform.Lower}, nil
	case "trim":
		return []pipeline.Stage{transform.Trim}, nil
	case "number":
		return []pipeline.Stage{transform.NumberLines()}, nil
	case "grep", "filter":
		if match == "" {
			return nil, fmt.Errorf("--match required for %s mode", strings.ToLower(strings.TrimSpace(mode)))
		}
		return []pipeline.Stage{transform.Contains(match)}, nil
	case "drop", "exclude":
		if match == "" {
			return nil, fmt.Errorf("--match required for %s mode", strings.ToLower(strings.TrimSpace(mode)))
		}
		return []pipeline.Stage{transform.NotContains(match)}, nil
	case "json-field", "json-extract":
		if field == "" {
			return nil, fmt.Errorf("--field required for %s mode", strings.ToLower(strings.TrimSpace(mode)))
		}
		return []pipeline.Stage{transform.JSONExtractField(field)}, nil
	case "json-pretty":
		return []pipeline.Stage{transform.JSONPretty}, nil
	case "replace":
		if oldStr == "" {
			return nil, fmt.Errorf("--old required for replace mode")
		}
		return []pipeline.Stage{transform.Replace(oldStr, newStr)}, nil
	case "dedup":
		return []pipeline.Stage{transform.Dedup()}, nil
	case "chain":
		return []pipeline.Stage{
			transform.Trim,
			transform.Dedup(),
			transform.NumberLines(),
		}, nil
	default:
		return nil, fmt.Errorf("unknown mode: %q", mode)
	}
}
