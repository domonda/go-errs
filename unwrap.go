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

// UnwrapCallStack unwraps callstack information from err
// and returns the first non callstack wrapper error.
// It does not remove callstack wrapping further down the
// wrapping chain if the top error
// is not wrapped with callstack information.
func UnwrapCallStack(err error) error {
	for p, ok := err.(callStackProvider); ok; p, ok = err.(callStackProvider) {
		err = p.Unwrap()
	}
	return err
}

// IsType returns if err or any unwrapped error
// is of the type of the passed ref error.
// It works similar than errors.As but
// without assigning to the ref error
// and without checking for Is or As methods.
func IsType(err, ref error) bool {
	if err == ref {
		return true
	}
	if err == nil {
		return false
	}
	t := reflect.TypeOf(ref)
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

// Type indicates if err is not nil and it
// or any unwrapped error is of the type T.
// It works similar than errors.As but
// without assigning to the ref error
// and without checking for Is or As methods.
func Type[T error](err error) bool {
	for err != nil {
		if _, ok := err.(T); ok {
			return true
		}
		err = errors.Unwrap(err)
	}
	return false
}
