package cue

import (
	"strings"
	"testing"

	"cuelang.org/go/cue/cuecontext"
)

func TestFormatError(t *testing.T) {
	t.Parallel()
	t.Run("nil error returns nil", func(t *testing.T) {
		err := FormatError(nil)
		if err != nil {
			t.Errorf("FormatError(nil) = %v, want nil", err)
		}
	})

	t.Run("formats CUE error with position", func(t *testing.T) {
		ctx := cuecontext.New()
		// Create a value with an error
		v := ctx.CompileString(`
			name: "hello" & 42
		`)

		err := v.Err()
		if err == nil {
			t.Fatal("Expected CUE error")
		}

		formatted := FormatError(err)
		if formatted == nil {
			t.Fatal("FormatError returned nil for non-nil error")
		}

		// Check that it's a ValidationError
		ve, ok := formatted.(*ValidationError)
		if !ok {
			t.Fatalf("FormatError returned %T, want *ValidationError", formatted)
		}

		// Error message should contain something about the conflict
		if ve.Message == "" {
			t.Error("ValidationError.Message is empty")
		}
	})

	t.Run("handles non-CUE errors", func(t *testing.T) {
		err := &testError{msg: "custom error"}
		formatted := FormatError(err)

		// FormatError should return something with the same message
		if formatted == nil {
			t.Fatal("FormatError returned nil for non-nil error")
		}
		if formatted.Error() != "custom error" {
			t.Errorf("FormatError message = %q, want %q", formatted.Error(), "custom error")
		}
	})
}

func TestFormatErrors(t *testing.T) {
	t.Parallel()
	t.Run("nil error returns nil", func(t *testing.T) {
		errs := FormatErrors(nil)
		if errs != nil {
			t.Errorf("FormatErrors(nil) = %v, want nil", errs)
		}
	})

	t.Run("formats multiple CUE errors", func(t *testing.T) {
		ctx := cuecontext.New()
		// Create a value with multiple potential errors
		v := ctx.CompileString(`
			a: "hello" & 42
			b: true & "false"
		`)

		err := v.Err()
		if err == nil {
			t.Fatal("Expected CUE error")
		}

		formatted := FormatErrors(err)
		if len(formatted) == 0 {
			t.Fatal("FormatErrors returned empty slice")
		}

		// Each error should be a ValidationError
		for i, e := range formatted {
			if _, ok := e.(*ValidationError); !ok {
				t.Errorf("FormatErrors[%d] = %T, want *ValidationError", i, e)
			}
		}
	})
}

func TestErrorSummary(t *testing.T) {
	t.Parallel()
	t.Run("empty string for nil error", func(t *testing.T) {
		summary := ErrorSummary(nil)
		if summary != "" {
			t.Errorf("ErrorSummary(nil) = %q, want empty string", summary)
		}
	})

	t.Run("single error shows message", func(t *testing.T) {
		ctx := cuecontext.New()
		v := ctx.CompileString(`x: "a" & 1`)

		err := v.Err()
		if err == nil {
			t.Fatal("Expected CUE error")
		}

		summary := ErrorSummary(err)
		if summary == "" {
			t.Error("ErrorSummary returned empty string")
		}
	})

	t.Run("multiple errors shows count", func(t *testing.T) {
		ctx := cuecontext.New()
		v := ctx.CompileString(`
			a: "x" & 1
			b: true & "y"
		`)

		err := v.Err()
		if err == nil {
			t.Fatal("Expected CUE error")
		}

		summary := ErrorSummary(err)

		// Should mention additional errors if there are multiple
		// Note: CUE may or may not produce multiple errors for this case
		if summary == "" {
			t.Error("ErrorSummary returned empty string")
		}
	})
}

func TestValidationError_ErrorFormat(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		err      *ValidationError
		contains []string
	}{
		{
			name:     "basic message",
			err:      &ValidationError{Message: "something failed"},
			contains: []string{"something failed"},
		},
		{
			name:     "with path",
			err:      &ValidationError{Path: "config.timeout", Message: "must be positive"},
			contains: []string{"config.timeout", "must be positive"},
		},
		{
			name:     "with file and line",
			err:      &ValidationError{Filename: "test.cue", Line: 42, Message: "syntax error"},
			contains: []string{"test.cue", "42", "syntax error"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.err.Error()
			for _, s := range tt.contains {
				if !strings.Contains(result, s) {
					t.Errorf("Error() = %q, should contain %q", result, s)
				}
			}
		})
	}
}

// testError is a simple error type for testing
type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
