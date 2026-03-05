package transform

import (
	"strings"
	"testing"
)

func TestUpper(t *testing.T) {
	if got := Upper("hello"); got != "HELLO" {
		t.Fatalf("Upper = %q, want HELLO", got)
	}
}

func TestLower(t *testing.T) {
	if got := Lower("HELLO"); got != "hello" {
		t.Fatalf("Lower = %q, want hello", got)
	}
}

func TestTrim(t *testing.T) {
	if got := Trim("  hello  "); got != "hello" {
		t.Fatalf("Trim = %q, want hello", got)
	}
}

func TestPrefixSuffix(t *testing.T) {
	prefix := Prefix("[x] ")
	suffix := Suffix(" !")

	if got := prefix("hello"); got != "[x] hello" {
		t.Fatalf("Prefix = %q", got)
	}
	if got := suffix("hello"); got != "hello !" {
		t.Fatalf("Suffix = %q", got)
	}
}

func TestNumberLines(t *testing.T) {
	fn := NumberLines()
	if got := fn("alpha"); got != "     1 | alpha" {
		t.Fatalf("first = %q", got)
	}
	if got := fn("beta"); got != "     2 | beta" {
		t.Fatalf("second = %q", got)
	}
}

func TestContains(t *testing.T) {
	fn := Contains("error")
	if got := fn("error: broke"); got == "" {
		t.Fatal("expected matching line to be kept")
	}
	if got := fn("info: fine"); got != "" {
		t.Fatal("expected non-matching line to be dropped")
	}
}

func TestNotContains(t *testing.T) {
	fn := NotContains("debug")
	if got := fn("debug: noise"); got != "" {
		t.Fatal("expected matching line to be dropped")
	}
	if got := fn("error: real"); got == "" {
		t.Fatal("expected non-matching line to be kept")
	}
}

func TestMatchRegex(t *testing.T) {
	fn, err := MatchRegex(`^\d{3}-`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if fn("404-not-found") == "" {
		t.Fatal("expected line to match")
	}
	if fn("info: ok") != "" {
		t.Fatal("expected non-matching line to be dropped")
	}
}

func TestMatchRegexInvalidPattern(t *testing.T) {
	if _, err := MatchRegex("["); err == nil {
		t.Fatal("expected compile error")
	}
}

func TestDedupConsecutive(t *testing.T) {
	fn := Dedup()
	if fn("a") == "" {
		t.Fatal("first line should not be dropped")
	}
	if fn("a") != "" {
		t.Fatal("duplicate should be dropped")
	}
	if fn("b") == "" {
		t.Fatal("new line should be kept")
	}
}

func TestDedupFirstEmptyLineKept(t *testing.T) {
	fn := Dedup()
	if got := fn(""); got != "" {
		t.Fatalf("first empty line should be kept as empty string, got %q", got)
	}
	if got := fn(""); got != "" {
		t.Fatalf("duplicate empty line should be dropped, got %q", got)
	}
}

func TestJSONExtractField(t *testing.T) {
	fn := JSONExtractField("name")

	tests := []struct {
		input string
		want  string
	}{
		{`{"name":"Nick","age":30}`, "Nick"},
		{`{"name":true}`, "true"},
		{`{"age":30}`, ""},
		{"not json", ""},
	}

	for _, tt := range tests {
		if got := fn(tt.input); got != tt.want {
			t.Fatalf("JSONExtractField(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestJSONPretty(t *testing.T) {
	got := JSONPretty(`{"name":"Nick","age":30}`)
	if !strings.Contains(got, "\n") {
		t.Fatalf("expected pretty output with newlines, got %q", got)
	}
	if !strings.Contains(got, "  \"name\"") {
		t.Fatalf("expected indented key, got %q", got)
	}
}

func TestJSONPrettyPassThroughOnInvalidJSON(t *testing.T) {
	input := "not-json"
	if got := JSONPretty(input); got != input {
		t.Fatalf("got %q, want %q", got, input)
	}
}

func TestReplace(t *testing.T) {
	fn := Replace("world", "Go")
	if got := fn("hello world"); got != "hello Go" {
		t.Fatalf("Replace = %q", got)
	}
}

func TestReplaceRegex(t *testing.T) {
	fn, err := ReplaceRegex(`\d+`, "#")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := fn("id=123"); got != "id=#" {
		t.Fatalf("ReplaceRegex = %q", got)
	}
}

func TestReplaceRegexInvalidPattern(t *testing.T) {
	if _, err := ReplaceRegex("[", "x"); err == nil {
		t.Fatal("expected compile error")
	}
}
