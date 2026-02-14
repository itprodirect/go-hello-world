package validator

import (
	"errors"
	"strings"
	"testing"
)

func TestValidateName_EmptyReturnsErrEmpty(t *testing.T) {
	err := ValidateName("   ")
	if err == nil {
		t.Fatal("expected error for blank name")
	}
	if !errors.Is(err, ErrEmpty) {
		t.Fatalf("expected ErrEmpty, got: %v", err)
	}
}

func TestValidateName_TooLongReturnsErrTooLong(t *testing.T) {
	long := strings.Repeat("a", 51)
	err := ValidateName(long)
	if !errors.Is(err, ErrTooLong) {
		t.Fatalf("expected ErrTooLong, got: %v", err)
	}
}

func TestValidateName_BadCharsReturnsErrBadChars(t *testing.T) {
	err := ValidateName("Nick<script>")
	if !errors.Is(err, ErrBadChars) {
		t.Fatalf("expected ErrBadChars, got: %v", err)
	}
}

func TestValidateName_ValidReturnsNil(t *testing.T) {
	if err := ValidateName("Nick"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateName_ErrorsAs_GetsField(t *testing.T) {
	err := ValidateName("")
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatal("expected ValidationError")
	}
	if ve.Field != "name" {
		t.Fatalf("Field = %q, want %q", ve.Field, "name")
	}
}

func TestValidateRepeat_OutOfRange(t *testing.T) {
	if err := ValidateRepeat(0); err == nil {
		t.Fatal("expected error for 0")
	}
	if err := ValidateRepeat(1001); err == nil {
		t.Fatal("expected error for 1001")
	}
}

func TestValidateRepeat_ValidRange(t *testing.T) {
	for _, n := range []int{1, 500, 1000} {
		if err := ValidateRepeat(n); err != nil {
			t.Fatalf("unexpected error for %d: %v", n, err)
		}
	}
}
