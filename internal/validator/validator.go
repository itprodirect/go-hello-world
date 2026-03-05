package validator

import (
	"strings"
	"unicode/utf8"

	"github.com/itprodirect/go-hello-world/internal/apperror"
)

const (
	maxNameLength = 50
	minRepeat     = 1
	maxRepeat     = 1000
)

// ValidateName checks a user-provided name that may be empty.
func ValidateName(name string) error {
	trimmed := strings.TrimSpace(name)

	if trimmed == "" {
		return nil
	}

	if utf8.RuneCountInString(trimmed) > maxNameLength {
		return apperror.NewFieldError("name", "must be 50 characters or fewer", apperror.ErrValidation)
	}

	for _, ch := range trimmed {
		if ch == '<' || ch == '>' || ch == '&' {
			return apperror.NewFieldError("name", "contains unsafe characters", apperror.ErrValidation)
		}
	}

	return nil
}

// ValidateRequiredName checks a required user-provided name.
func ValidateRequiredName(name string) error {
	if strings.TrimSpace(name) == "" {
		return apperror.NewFieldError("name", "must not be empty", apperror.ErrValidation)
	}
	return ValidateName(name)
}

// ValidateRepeat checks the allowed repeat range.
func ValidateRepeat(n int) error {
	if n < minRepeat || n > maxRepeat {
		return apperror.NewFieldError("repeat", "must be 1-1000", apperror.ErrValidation)
	}
	return nil
}
