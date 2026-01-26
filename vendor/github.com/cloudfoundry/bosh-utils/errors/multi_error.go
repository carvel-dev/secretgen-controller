package errors

import (
	"strings"
)

type MultiError struct {
	Errors []error
}

func NewMultiError(errors ...error) error {
	return MultiError{Errors: errors}
}

func (e MultiError) Error() string {
	errors := make([]string, len(e.Errors), len(e.Errors)) //nolint:gosimple
	for i, err := range e.Errors {
		errors[i] = err.Error()
	}
	return strings.Join(errors, "\n")
}
