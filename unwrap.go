package errs

import (
	"errors"
	"reflect"
)

// Root unwraps err recursively and returns the root error.
func Root(err error) error {
	for {
		unwrapped := errors.Unwrap(err)
		if unwrapped == nil {
			return err
		}
		err = unwrapped
	}
}

// IsType returns if err or any unwrapped error
// is of the type of the passed target error.
// It returns the same result as errors.As
// without assigning to the target error.
func IsType(err, target error) bool {
	if err == target {
		return true
	}
	if err == nil {
		return false
	}
	t := reflect.TypeOf(target)
	for {
		if reflect.TypeOf(err) == t {
			return true
		}
		err = errors.Unwrap(err)
		if err == nil {
			return false
		}
	}
}
