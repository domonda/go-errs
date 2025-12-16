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

// Has is a shortcut for errors.As
// when the target error value is not needed.
func Has[T error](err error) bool {
	var target T
	return errors.As(err, &target)
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
	targetType := reflect.TypeOf((*T)(nil)).Elem()
	for {
		var target T
		if reflect.TypeOf(err).AssignableTo(targetType) {
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

// UnwrapCallStack removes all top-level callstack wrapping from err
// and returns the underlying error without the callstack information.
//
// Unlike Root which unwraps to the root cause, this function only removes
// callstack wrappers (including those with function parameters) but preserves
// the error chain.
//
// This is useful when you want to compare or match errors without
// the callstack information affecting the comparison, or when you need
// to access the wrapped error while discarding debug information.
//
// Examples:
//
//	// Remove callstack wrapper from error
//	err := errs.New("something failed")
//	cleaned := errs.UnwrapCallStack(err)
//	// cleaned is the underlying Sentinel without callstack
//
//	// Compare errors without callstack
//	err1 := errs.WrapWithCallStack(sentinel)
//	err2 := errs.WrapWithCallStack(sentinel)
//	// err1 != err2 (different callstacks)
//	// errs.UnwrapCallStack(err1) == errs.UnwrapCallStack(err2) == sentinel
//
//	// Preserve error chain while removing top-level callstack
//	wrapped := fmt.Errorf("context: %w", sentinel)
//	withStack := errs.WrapWithCallStack(wrapped)
//	result := errs.UnwrapCallStack(withStack)
//	// result == wrapped (still wraps sentinel)
//
// Note: This only removes top-level callstack wrapping. If there are
// callstack wrappers further down the error chain, they are preserved.
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
		switch x := err.(type) {
		case interface{ Unwrap() error }:
			err = x.Unwrap()
		case interface{ Unwrap() []error }:
			for _, e := range x.Unwrap() {
				if Type[T](e) {
					return true
				}
			}
			return false
		default:
			return false
		}
	}
	return false
}
