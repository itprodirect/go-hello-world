package greeter

import (
	"fmt"
	"strings"
)

// Greeter defines greeting strategy behavior.
type Greeter interface {
	Greet(name string, sequence int) string
}

// Standard is the default greeting style.
type Standard struct{}

func (g Standard) Greet(name string, sequence int) string {
	return buildMsg(name, sequence, "Hello, %s! (#%d)", "Hello, %s!")
}

// Formal is a polite greeting style.
type Formal struct{}

func (g Formal) Greet(name string, sequence int) string {
	return buildMsg(name, sequence, "Good day, %s. [#%d]", "Good day, %s.")
}

// Shout is an emphatic greeting style.
type Shout struct{}

func (g Shout) Greet(name string, sequence int) string {
	base := buildMsg(name, sequence, "HEY %s!!! (#%d)", "HEY %s!!!")
	return strings.ToUpper(base)
}

// New returns a greeter implementation based on style.
func New(style string) Greeter {
	switch strings.ToLower(strings.TrimSpace(style)) {
	case "formal":
		return Formal{}
	case "shout":
		return Shout{}
	default:
		return Standard{}
	}
}

// BuildGreeting preserves backwards compatibility with the original API.
func BuildGreeting(name string, sequence int) string {
	return Standard{}.Greet(name, sequence)
}

func buildMsg(name string, sequence int, withSeq, withoutSeq string) string {
	cleanName := strings.TrimSpace(name)
	if cleanName == "" {
		cleanName = "world"
	}

	if sequence > 0 {
		return fmt.Sprintf(withSeq, cleanName, sequence)
	}
	return fmt.Sprintf(withoutSeq, cleanName)
}
