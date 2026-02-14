package greeter

import (
	"fmt"
	"strings"
)

// BuildGreeting builds a greeting and optionally includes sequence information.
func BuildGreeting(name string, sequence int) string {
	cleanName := strings.TrimSpace(name)
	if cleanName == "" {
		cleanName = "world"
	}

	if sequence > 0 {
		return fmt.Sprintf("Hello, %s! (#%d)", cleanName, sequence)
	}

	return fmt.Sprintf("Hello, %s!", cleanName)
}
