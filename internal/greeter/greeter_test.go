package greeter

import "testing"

func TestBuildGreetingWithSequence(t *testing.T) {
	got := BuildGreeting("Nick", 3)
	want := "Hello, Nick! (#3)"

	if got != want {
		t.Fatalf("BuildGreeting() = %q, want %q", got, want)
	}
}

func TestBuildGreetingFallsBackToWorld(t *testing.T) {
	got := BuildGreeting("   ", 0)
	want := "Hello, world!"

	if got != want {
		t.Fatalf("BuildGreeting() = %q, want %q", got, want)
	}
}
