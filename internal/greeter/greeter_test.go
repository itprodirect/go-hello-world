package greeter

import (
	"strings"
	"testing"
)

func TestBuildGreetingWithSequence(t *testing.T) {
	got := BuildGreeting("Nick", 3)
	want := "Hello, Nick! (#3)"

	if got != want {
		t.Fatalf("BuildGreeting() = %q, want %q", got, want)
	}
}

func TestGreeterStyles(t *testing.T) {
	tests := []struct {
		name     string
		style    string
		input    string
		sequence int
		contains string
	}{
		{name: "standard", style: "standard", input: "Nick", sequence: 1, contains: "Hello, Nick!"},
		{name: "formal", style: "formal", input: "Nick", sequence: 1, contains: "Good day, Nick."},
		{name: "shout", style: "shout", input: "Nick", sequence: 1, contains: "HEY NICK!!!"},
		{name: "fallback", style: "unknown", input: "Nick", sequence: 1, contains: "Hello, Nick!"},
		{name: "blank-name", style: "formal", input: "", sequence: 0, contains: "Good day, world."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := New(tt.style).Greet(tt.input, tt.sequence)
			if !strings.Contains(got, tt.contains) {
				t.Fatalf("New(%q).Greet() = %q, want contains %q", tt.style, got, tt.contains)
			}
		})
	}
}

var (
	_ Greeter = Standard{}
	_ Greeter = Formal{}
	_ Greeter = Shout{}
)

func TestBuildGreetingFallsBackToWorld(t *testing.T) {
	got := BuildGreeting("   ", 0)
	want := "Hello, world!"

	if got != want {
		t.Fatalf("BuildGreeting() = %q, want %q", got, want)
	}
}
