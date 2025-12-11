package cue

import (
	"fmt"
	"strconv"

	"cuelang.org/go/cue/errors"
)

// FormatError converts a CUE error into a user-friendly error.
// It extracts position information and formats messages for clarity.
func FormatError(err error) error {
	if err == nil {
		return nil
	}

	// Try to extract CUE errors with position information
	cueErrs := errors.Errors(err)
	if len(cueErrs) == 0 {
		return err
	}

	// Format the first error with position info
	first := cueErrs[0]
	pos := first.Position()

	// Use Msg() to get the unformatted message without position prefix.
	// Msg() returns the format string and args separately, allowing us to
	// reconstruct the message without file:line prefixes.
	format, args := first.Msg()
	var message string
	if format != "" && len(args) > 0 {
		message = fmt.Sprintf(format, args...)
	}
	// Fallback to Error() if Msg() returns empty or unusable format
	if message == "" {
		message = first.Error()
	}

	ve := &ValidationError{
		Message: message,
	}

	if pos.IsValid() {
		ve.Filename = pos.Filename()
		ve.Line = pos.Line()
		ve.Column = pos.Column()
	}

	return ve
}

// FormatErrors formats multiple CUE errors into a single error message.
func FormatErrors(err error) []error {
	if err == nil {
		return nil
	}

	cueErrs := errors.Errors(err)
	if len(cueErrs) == 0 {
		return []error{err}
	}

	result := make([]error, 0, len(cueErrs))
	for _, e := range cueErrs {
		result = append(result, FormatError(e))
	}
	return result
}

// ErrorSummary returns a short summary of CUE errors.
func ErrorSummary(err error) string {
	if err == nil {
		return ""
	}

	cueErrs := errors.Errors(err)
	if len(cueErrs) == 0 {
		return err.Error()
	}

	if len(cueErrs) == 1 {
		return FormatError(cueErrs[0]).Error()
	}

	// Multiple errors: show first and count
	first := FormatError(cueErrs[0]).Error()
	return first + " (and " + strconv.Itoa(len(cueErrs)-1) + " more errors)"
}
