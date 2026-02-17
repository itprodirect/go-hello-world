package apperror

import (
	"errors"
	"testing"
)

func TestFieldErrorErrorString(t *testing.T) {
	tests := []struct {
		name string
		fe   *FieldError
		want string
	}{
		{
			name: "with sentinel",
			fe:   NewFieldError("name", "must not be empty", ErrValidation),
			want: "name: must not be empty: validation failed",
		},
		{
			name: "without sentinel",
			fe:   &FieldError{Field: "age", Message: "required"},
			want: "age: required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.fe.Error(); got != tt.want {
				t.Errorf("Error() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFieldErrorUnwrap(t *testing.T) {
	fe := NewFieldError("url", "bad format", ErrValidation)
	if !errors.Is(fe, ErrValidation) {
		t.Fatal("errors.Is should match ErrValidation")
	}

	var target *FieldError
	if !errors.As(fe, &target) {
		t.Fatal("errors.As should extract *FieldError")
	}
	if target.Field != "url" {
		t.Errorf("Field = %q, want %q", target.Field, "url")
	}
}

func TestWrapNilPassthrough(t *testing.T) {
	if got := Wrap(nil, "context"); got != nil {
		t.Fatalf("Wrap(nil) should be nil, got %v", got)
	}
}

func TestWrapPreservesSentinel(t *testing.T) {
	wrapped := Wrap(ErrTimeout, "checking endpoint")
	if !errors.Is(wrapped, ErrTimeout) {
		t.Fatal("wrapped error should still match ErrTimeout")
	}
}

func TestConvenienceChecks(t *testing.T) {
	tests := []struct {
		name  string
		err   error
		check func(error) bool
		want  bool
	}{
		{"IsNotFound true", ErrNotFound, IsNotFound, true},
		{"IsNotFound false", ErrValidation, IsNotFound, false},
		{"IsValidation true", NewFieldError("x", "y", ErrValidation), IsValidation, true},
		{"IsTimeout true", Wrap(ErrTimeout, "ctx"), IsTimeout, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.check(tt.err); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}
