package transform

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/itprodirect/go-hello-world/internal/pipeline"
)

// Upper converts to uppercase.
func Upper(line string) string {
	return strings.ToUpper(line)
}

// Lower converts to lowercase.
func Lower(line string) string {
	return strings.ToLower(line)
}

// Trim removes leading and trailing whitespace.
func Trim(line string) string {
	return strings.TrimSpace(line)
}

// Prefix adds a prefix to every line.
func Prefix(prefix string) pipeline.Stage {
	return func(line string) string {
		return prefix + line
	}
}

// Suffix adds a suffix to every line.
func Suffix(suffix string) pipeline.Stage {
	return func(line string) string {
		return line + suffix
	}
}

// NumberLines adds line numbers and keeps state in a closure.
func NumberLines() pipeline.Stage {
	n := 0
	return func(line string) string {
		n++
		return fmt.Sprintf("%6d | %s", n, line)
	}
}

// Contains keeps lines containing substr.
func Contains(substr string) pipeline.Stage {
	return func(line string) string {
		if strings.Contains(line, substr) {
			return line
		}
		return ""
	}
}

// NotContains drops lines containing substr.
func NotContains(substr string) pipeline.Stage {
	return func(line string) string {
		if strings.Contains(line, substr) {
			return ""
		}
		return line
	}
}

// MatchRegex keeps lines matching the regex pattern.
func MatchRegex(pattern string) (pipeline.Stage, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("compile regex: %w", err)
	}

	return func(line string) string {
		if re.MatchString(line) {
			return line
		}
		return ""
	}, nil
}

// Dedup drops consecutive duplicate lines.
func Dedup() pipeline.Stage {
	prev := ""
	initialized := false

	return func(line string) string {
		if initialized && line == prev {
			return ""
		}

		initialized = true
		prev = line
		return line
	}
}

// JSONExtractField extracts a field value from JSON lines.
// Missing fields or invalid JSON are dropped.
func JSONExtractField(field string) pipeline.Stage {
	return func(line string) string {
		var obj map[string]interface{}
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			return ""
		}

		value, ok := obj[field]
		if !ok {
			return ""
		}

		return fmt.Sprintf("%v", value)
	}
}

// JSONPretty pretty-prints JSON lines. Non-JSON input passes through unchanged.
func JSONPretty(line string) string {
	var obj interface{}
	if err := json.Unmarshal([]byte(line), &obj); err != nil {
		return line
	}

	pretty, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return line
	}

	return string(pretty)
}

// Replace does a simple string replacement.
func Replace(old, new string) pipeline.Stage {
	return func(line string) string {
		return strings.ReplaceAll(line, old, new)
	}
}

// ReplaceRegex does regex replacement.
func ReplaceRegex(pattern, replacement string) (pipeline.Stage, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return nil, fmt.Errorf("compile regex: %w", err)
	}

	return func(line string) string {
		return re.ReplaceAllString(line, replacement)
	}, nil
}
