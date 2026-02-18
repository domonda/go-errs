package errs

import (
	"errors"
	"reflect"
)

// Root unwraps err recursively and returns the root error.
//
// It uses the interfaces
//
//	interface{ Unwrap() error }
//	interface{ Unwrap() []error }
//
// For multi-error trees (errors.Join), it returns the root
// of the first non-nil branch.
func Root(err error) error {
	for err != nil {
		switch x := err.(type) {
		case interface{ Unwrap() error }:
			unwrapped := x.Unwrap()
			if unwrapped == nil {
				return err
			}
			err = unwrapped
		case interface{ Unwrap() []error }:
			for _, e := range x.Unwrap() {
				if e != nil {
					return Root(e)
				}
			}
			return err
		default:
			return err
		}
	}
	return err
}

// Has reports whether err's tree contains an error of type T.
// It is a shortcut for errors.As when the target value is not needed.
//
// Since Go 1.26, the standard library provides [errors.AsType]
// which returns the matched error value along with a bool:
//
//	target, ok := errors.AsType[*MyError](err)
//
// Use Has when you only need the boolean check and don't need
// the matched error value. Use errors.AsType when you need both.
func Has[T error](err error) bool {
	var target T
	return errors.As(err, &target)
}

// As returns all errors of type T in the wrapping tree of err.
//
// Unlike [errors.AsType] (Go 1.26+) which returns only the first match,
// this function traverses the full error tree and collects all matches.
// This is particularly useful with multi-errors ([errors.Join]) where
// multiple errors of the same type may exist in different branches.
//
// Example:
//
//	err := errors.Join(
//	    &ValidationError{Field: "name"},
//	    &ValidationError{Field: "email"},
//	)
//	// errors.AsType returns only the first:
//	first, _ := errors.AsType[*ValidationError](err) // Field: "name"
//	// errs.As returns all:
//	all := errs.As[*ValidationError](err)             // both "name" and "email"
//
// It traverses the full tree using the interface methods:
//
//	Unwrap() error
//	Unwrap() []error
//
// An error err matches the type T if the type assertion err.(T) holds,
// or if the error has a method As(any) bool such that err.As(target)
// returns true when target is a non-nil *T. In the latter case, the As
// method is responsible for setting target.
func As[T error](err error) []T {
	if err == nil {
		return nil
	}
	var errs []T
	as(err, &errs)
	return errs
}

func as[T error](err error, errs *[]T) {
	for {
		if e, ok := err.(T); ok {
			*errs = append(*errs, e)
		} else if x, ok := err.(interface{ As(any) bool }); ok {
			var target T
			if x.As(&target) {
				*errs = append(*errs, target)
			}
		}
		switch x := err.(type) {
		case interface{ Unwrap() error }:
			err = x.Unwrap()
			if err == nil {
				return
			}
		case interface{ Unwrap() []error }:
			for _, err := range x.Unwrap() {
				if err == nil {
					continue
				}
				as(err, errs)
			}
			return
		default:
			return
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
// It works similar to errors.As but
// without assigning to the ref error
// and without checking for Is or As methods.
//
// It uses the interfaces
//
//	interface{ Unwrap() error }
//	interface{ Unwrap() []error }
func IsType(err, ref error) bool {
	if err == ref {
		return true
	}
	if err == nil {
		return false
	}
	t := reflect.TypeOf(ref)
	for err != nil {
		if reflect.TypeOf(err) == t {
			return true
		}
		switch x := err.(type) {
		case interface{ Unwrap() error }:
			err = x.Unwrap()
		case interface{ Unwrap() []error }:
			for _, e := range x.Unwrap() {
				if IsType(e, ref) {
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

// Type indicates if err is not nil and it
// or any unwrapped error is of the type T.
// It works similarly to errors.As but
// without assigning to a target error
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
