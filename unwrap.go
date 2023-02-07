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

// As returns all errors of type T in the wrapping tree of err.
//
// This function is similar to errors.As
// but traverses the full tree using the interface methods:
//
//	Unwrap() error
//	Unwrap() []error
func As[T error](err error) []T {
	if err == nil {
		return nil
	}
	var errs []T
	var target T
	targetType := reflect.TypeOf(target)
	for {
		if reflect.TypeOf(err).AssignableTo(targetType) {
			var target T
			reflect.ValueOf(&target).Elem().Set(reflect.ValueOf(err))
			errs = append(errs, target)
		}
		if x, ok := err.(interface{ As(any) bool }); ok && x.As(&target) {
			errs = append(errs, target)
		}
		switch x := err.(type) {
		case interface{ Unwrap() error }:
			err = x.Unwrap()
			if err == nil {
				return errs
			}
		case interface{ Unwrap() []error }:
			for _, err := range x.Unwrap() {
				errs = append(errs, As[T](err)...)
			}
			return errs
		default:
			return errs
		}
	}
}

// // UnwrapAll returns all wrapped errors
// // not including the wrapper errors.
// //
// // It uses the interfaces
// //
// //	interface{ Unwrap() error }
// //	interface{ Unwrap() []error }
// func UnwrapAll(err error) []error {
// 	if err == nil {
// 		return nil
// 	}
// 	var errs []error
// 	for {
// 		switch x := err.(type) {
// 		case interface{ Unwrap() error }:
// 			err = x.Unwrap()
// 			if err == nil {
// 				return errs
// 			}
// 			errs = append(errs, err)

// 		case interface{ Unwrap() []error }:
// 			for _, e := range x.Unwrap() {
// 				errs = append(errs, UnwrapAll(e)...)
// 			}
// 			return errs

// 		default:
// 			return append(errs, err)
// 		}
// 	}
// }

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
