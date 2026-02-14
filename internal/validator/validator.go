package validator

import (
	"errors"
	"fmt"
	"strings"
)

var (
	ErrEmpty    = errors.New("value must not be empty")
	ErrTooLong  = errors.New("value exceeds maximum length")
	ErrBadChars = errors.New("value contains disallowed characters")
)

// ValidationError wraps a validation issue with the field name.
type ValidationError struct {
	Field string
	Err   error
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation failed on %q: %v", e.Field, e.Err)
}

// Unwrap allows errors.Is and errors.As to inspect the wrapped error.
func (e *ValidationError) Unwrap() error {
	return e.Err
}

// ValidateName checks a user-provided name.
func ValidateName(name string) error {
	trimmed := strings.TrimSpace(name)

	if trimmed == "" {
		return &ValidationError{Field: "name", Err: ErrEmpty}
	}

	if len(trimmed) > 50 {
		return &ValidationError{
			Field: "name",
			Err:   fmt.Errorf("%w: got %d chars", ErrTooLong, len(trimmed)),
		}
	}

	for _, ch := range trimmed {
		if ch == '<' || ch == '>' || ch == '&' {
			return &ValidationError{
				Field: "name",
				Err:   fmt.Errorf("%w: %q", ErrBadChars, string(ch)),
			}
		}
	}

	return nil
}

// ValidateRepeat checks the allowed repeat range.
func ValidateRepeat(n int) error {
	if n < 1 {
		return &ValidationError{
			Field: "repeat",
			Err:   fmt.Errorf("must be >= 1, got %d", n),
		}
	}
	if n > 1000 {
		return &ValidationError{
			Field: "repeat",
			Err:   fmt.Errorf("must be <= 1000, got %d", n),
		}
	}
	return nil
}
