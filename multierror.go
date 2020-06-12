package errs

import (
	"errors"
	"strings"
)

// MultiError combines multiple errors into one.
// The Error method returns the strings from the individual Error methods
// joined by the new line character '\n'.
// The motivation behind Combine and MultiError is to combine different
// logical errors into one, as compared to error wrapping,
// which adds more information to one logical error.
type MultiError interface {
	Error() string
	Errors() []error
}

type multiError []error

func (m multiError) Error() string {
	var b strings.Builder
	for i, err := range m {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(err.Error())
	}
	return b.String()
}

func (m multiError) Errors() []error {
	return m
}

func (m multiError) Is(target error) bool {
	for _, err := range m {
		if errors.Is(err, target) {
			return true
		}
	}
	return false
}

func (m multiError) As(target interface{}) bool {
	for _, err := range m {
		if errors.As(err, target) {
			return true
		}
	}
	return false
}
