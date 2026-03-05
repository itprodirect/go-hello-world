package validator

import (
	"errors"
	"strings"
	"testing"

	"github.com/itprodirect/go-hello-world/internal/apperror"
)

func TestValidateName_EmptyAllowed(t *testing.T) {
	for _, input := range []string{"", "   "} {
		if err := ValidateName(input); err != nil {
			t.Fatalf("ValidateName(%q) unexpected error: %v", input, err)
		}
	}
}

func TestValidateRequiredName_EmptyRejected(t *testing.T) {
	err := ValidateRequiredName("\t")
	if err == nil {
		t.Fatal("expected error for blank required name")
	}
	if !errors.Is(err, apperror.ErrValidation) {
		t.Fatalf("expected ErrValidation, got: %v", err)
	}

	var fe *apperror.FieldError
	if !errors.As(err, &fe) {
		t.Fatalf("expected FieldError, got: %T", err)
	}
	if fe.Field != "name" {
		t.Fatalf("Field=%q, want name", fe.Field)
	}
}

func TestValidateName_TooLongRuneCount(t *testing.T) {
	withinLimit := strings.Repeat("\u00E9", 50)
	if err := ValidateName(withinLimit); err != nil {
		t.Fatalf("unexpected error for 50-rune name: %v", err)
	}

	overLimit := strings.Repeat("\u00E9", 51)
	err := ValidateName(overLimit)
	if err == nil {
		t.Fatal("expected error for over-limit name")
	}
	if !errors.Is(err, apperror.ErrValidation) {
		t.Fatalf("expected ErrValidation, got: %v", err)
	}
}

func TestValidateName_BadCharsReturnsValidationError(t *testing.T) {
	err := ValidateName("Nick<script>")
	if err == nil {
		t.Fatal("expected error for unsafe name")
	}
	if !errors.Is(err, apperror.ErrValidation) {
		t.Fatalf("expected ErrValidation, got: %v", err)
	}
}

func TestValidateName_ValidReturnsNil(t *testing.T) {
	if err := ValidateName("Nick"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRepeat_OutOfRange(t *testing.T) {
	for _, n := range []int{0, 1001} {
		err := ValidateRepeat(n)
		if err == nil {
			t.Fatalf("expected error for %d", n)
		}
		if !errors.Is(err, apperror.ErrValidation) {
			t.Fatalf("expected ErrValidation for %d, got: %v", n, err)
		}
	}
}

func TestValidateRepeat_ValidRange(t *testing.T) {
	for _, n := range []int{1, 500, 1000} {
		if err := ValidateRepeat(n); err != nil {
			t.Fatalf("unexpected error for %d: %v", n, err)
		}
	}
}
