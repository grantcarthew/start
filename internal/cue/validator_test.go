package cue

import (
	"strings"
	"testing"

	"cuelang.org/go/cue/cuecontext"
)

func TestNewValidator(t *testing.T) {
	v := NewValidator()
	if v == nil {
		t.Fatal("NewValidator() returned nil")
	}
}

func TestValidator_Validate(t *testing.T) {
	ctx := cuecontext.New()

	t.Run("valid concrete value", func(t *testing.T) {
		value := ctx.CompileString(`
			name: "test"
			count: 42
			enabled: true
		`)

		v := NewValidator()
		err := v.Validate(value)
		if err != nil {
			t.Errorf("Validate() error = %v, want nil", err)
		}
	})

	t.Run("valid concrete value with Concrete option", func(t *testing.T) {
		value := ctx.CompileString(`
			name: "test"
			count: 42
		`)

		v := NewValidator()
		err := v.Validate(value, Concrete(true))
		if err != nil {
			t.Errorf("Validate(Concrete(true)) error = %v, want nil", err)
		}
	})

	t.Run("non-concrete value without Concrete option", func(t *testing.T) {
		value := ctx.CompileString(`
			name: string
			count: int
		`)

		v := NewValidator()
		// Without Concrete(true), non-concrete values are allowed
		err := v.Validate(value)
		if err != nil {
			t.Errorf("Validate() error = %v, want nil", err)
		}
	})

	t.Run("non-concrete value with Concrete option", func(t *testing.T) {
		value := ctx.CompileString(`
			name: string
			count: int
		`)

		v := NewValidator()
		err := v.Validate(value, Concrete(true))
		if err == nil {
			t.Error("Validate(Concrete(true)) should return error for non-concrete value")
		}
	})

	t.Run("structural error", func(t *testing.T) {
		value := ctx.CompileString(`
			name: "hello" & 42
		`)

		v := NewValidator()
		err := v.Validate(value)
		if err == nil {
			t.Error("Validate() should return error for conflicting values")
		}
	})
}

func TestValidator_ValidatePath(t *testing.T) {
	ctx := cuecontext.New()

	t.Run("existing path", func(t *testing.T) {
		value := ctx.CompileString(`
			config: {
				name: "test"
				count: 42
			}
		`)

		v := NewValidator()
		err := v.ValidatePath(value, "config.name")
		if err != nil {
			t.Errorf("ValidatePath() error = %v, want nil", err)
		}
	})

	t.Run("non-existing path", func(t *testing.T) {
		value := ctx.CompileString(`
			config: {
				name: "test"
			}
		`)

		v := NewValidator()
		err := v.ValidatePath(value, "config.missing")
		if err == nil {
			t.Error("ValidatePath() should return error for non-existing path")
		}

		// Check error message
		if !strings.Contains(err.Error(), "does not exist") {
			t.Errorf("error message should contain 'does not exist', got: %v", err)
		}
	})

	t.Run("path with Concrete option", func(t *testing.T) {
		value := ctx.CompileString(`
			config: {
				name: string
			}
		`)

		v := NewValidator()
		err := v.ValidatePath(value, "config.name", Concrete(true))
		if err == nil {
			t.Error("ValidatePath(Concrete(true)) should return error for non-concrete value")
		}
	})
}

func TestValidator_Exists(t *testing.T) {
	ctx := cuecontext.New()

	value := ctx.CompileString(`
		config: {
			name: "test"
			nested: {
				value: 123
			}
		}
	`)

	v := NewValidator()

	tests := []struct {
		path   string
		expect bool
	}{
		{"config", true},
		{"config.name", true},
		{"config.nested", true},
		{"config.nested.value", true},
		{"config.missing", false},
		{"missing", false},
		{"config.nested.missing", false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := v.Exists(value, tt.path)
			if got != tt.expect {
				t.Errorf("Exists(%q) = %v, want %v", tt.path, got, tt.expect)
			}
		})
	}
}

func TestValidationError_Error(t *testing.T) {
	tests := []struct {
		name   string
		err    *ValidationError
		expect string
	}{
		{
			name:   "message only",
			err:    &ValidationError{Message: "something went wrong"},
			expect: "something went wrong",
		},
		{
			name:   "with path",
			err:    &ValidationError{Path: "config.name", Message: "invalid value"},
			expect: "config.name: invalid value",
		},
		{
			name:   "with filename",
			err:    &ValidationError{Filename: "config.cue", Message: "syntax error"},
			expect: "config.cue: syntax error",
		},
		{
			name:   "with filename and line",
			err:    &ValidationError{Filename: "config.cue", Line: 10, Message: "unexpected token"},
			expect: "config.cue:10: unexpected token",
		},
		{
			name:   "filename takes precedence over path",
			err:    &ValidationError{Filename: "config.cue", Path: "x.y", Message: "error"},
			expect: "config.cue: error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.err.Error()
			if got != tt.expect {
				t.Errorf("Error() = %q, want %q", got, tt.expect)
			}
		})
	}
}
