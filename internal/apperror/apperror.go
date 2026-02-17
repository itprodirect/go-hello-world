package apperror

import (
	"errors"
	"fmt"
)

var (
	ErrNotFound    = errors.New("not found")
	ErrValidation  = errors.New("validation failed")
	ErrTimeout     = errors.New("operation timed out")
	ErrUnavailable = errors.New("service unavailable")
)

// FieldError carries field-level context while preserving sentinel matching.
type FieldError struct {
	Field   string
	Message string
	Err     error
}

func (e *FieldError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Field, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

func (e *FieldError) Unwrap() error {
	return e.Err
}

func NewFieldError(field, message string, sentinel error) *FieldError {
	return &FieldError{
		Field:   field,
		Message: message,
		Err:     sentinel,
	}
}

func Wrap(err error, message string) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", message, err)
}

func IsNotFound(err error) bool {
	return errors.Is(err, ErrNotFound)
}

func IsValidation(err error) bool {
	return errors.Is(err, ErrValidation)
}

func IsTimeout(err error) bool {
	return errors.Is(err, ErrTimeout)
}
