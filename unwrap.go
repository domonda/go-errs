package errs

import (
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strings"
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
//
// Has is kept only for compatibility reasons,
// consider using [errors.AsType] from the standard library
// (since Go 1.26) instead, which also returns the matched error value:
//
//	target, ok := errors.AsType[*MyError](err)
func Has[T error](err error) bool {
	_, ok := errors.AsType[T](err)
	return ok
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

// MessageWithoutCallStacks returns the message of err with every call stack
// and function parameter block of this package removed, at every level
// of the error tree. It returns "" for a nil err and "<nil>",
// like fmt, for a nil pointer whose Error method panics.
//
// The Error method of errors created by New, Errorf, WrapWithCallStack and
// WrapWithFuncParams appends the call stack and function parameters to the
// message. That is right for logs, but leaks source paths, line numbers and
// function arguments when the message is handed to a client, for example
// in an HTTP response, an MCP tool result or a validation message.
// Use MessageWithoutCallStacks for messages that leave the service:
//
//	func ValidateQuery(query string) (err error) {
//	    defer errs.WrapWithFuncParams(&err, query)
//
//	    if strings.Contains(query, "set_config") {
//	        return errs.New("query contains disallowed identifier: set_config")
//	    }
//	    return nil
//	}
//
//	if err := ValidateQuery(query); err != nil {
//	    err = errs.Errorf("invalid query: %w", err)
//	    log.Println(err) // message with both call stacks and the query parameter
//	    http.Error(w, errs.MessageWithoutCallStacks(err), http.StatusBadRequest)
//	    // "invalid query: query contains disallowed identifier: set_config"
//	}
//
// Which function to use:
//   - UnwrapCallStack removes only the outermost call-stack wrappers and
//     returns an error whose message can still contain the call stacks of
//     wrapped errors.
//   - Root returns the innermost error and drops the context
//     that wrapping errors added to the message.
//   - MessageWithoutCallStacks keeps the full message including all context
//     and removes only the call stacks.
//
// fmt.Errorf (and Errorf) render the Error result of a %w operand into their
// message when they are called, so the call stack of a wrapped error ends up
// in the middle of the wrapping error's message. MessageWithoutCallStacks
// removes the outermost call-stack wrappers, then replaces every occurrence
// of the Error result of each wrapped error (from Unwrap() error or
// Unwrap() []error) in the message with the wrapped error's message without
// call stacks, recursively. This handles fmt.Errorf with one or more %w verbs
// and errors.Join. The call-stack wrappers of this package render their
// message once and return the same text afterwards, so it matches what
// a wrapping error copied, even if a function parameter such as
// a context.Context changed since. The debug stack of a recovered panic
// (AsErrorWithDebugStack, RecoverPanicAsError,
// RecoverPanicAsErrorWithFuncParams) is removed as well.
//
// Call stack text that is not reachable by unwrapping stays in the message:
//   - an error formatted with %v or %s instead of %w,
//     for example Errorf("context: %v", err), so wrap errors with %w;
//   - a wrapper whose Error method does not contain the Error result
//     of its wrapped error verbatim, for example one that reformats it.
func MessageWithoutCallStacks(err error) string {
	if isNilPointer(err) {
		// A nil pointer's Error method usually panics,
		// return what fmt renders for it, which is "<nil>" in that case
		return fmt.Sprint(err)
	}
	err = UnwrapCallStack(err)
	if err == nil {
		return ""
	}
	return removeWrappedCallStacks(err, err.Error())
}

// messageWithoutCallStacksProvider is implemented by error types
// of this package that render call stack text into their message
// without a call-stack wrapper that MessageWithoutCallStacks could unwrap.
type messageWithoutCallStacksProvider interface {
	messageWithoutCallStacks() string
}

// removeWrappedCallStacks returns msg, the Error result of err,
// with the call stacks of the errors wrapped by err removed.
// err must not be a call-stack wrapper.
func removeWrappedCallStacks(err error, msg string) string {
	if p, ok := err.(messageWithoutCallStacksProvider); ok {
		return p.messageWithoutCallStacks()
	}
	var wrapped []error
	switch x := err.(type) {
	case interface{ Unwrap() error }:
		wrapped = []error{x.Unwrap()}
	case interface{ Unwrap() []error }:
		wrapped = x.Unwrap()
	}
	// Each wrapped error's message with and without call stacks
	var replacements [][2]string
	for _, w := range wrapped {
		if w == nil || isNilPointer(w) {
			// fmt renders a nil pointer error as "<nil>" (it recovers
			// the panic of its Error method), so there is nothing to replace
			continue
		}
		withStacks := w.Error()
		var withoutStacks string
		if _, isCallStack := w.(callStackProvider); isCallStack {
			withoutStacks = MessageWithoutCallStacks(w)
		} else {
			withoutStacks = removeWrappedCallStacks(w, withStacks)
		}
		if withStacks != withoutStacks {
			replacements = append(replacements, [2]string{withStacks, withoutStacks})
		}
	}
	if len(replacements) == 0 {
		return msg
	}
	// strings.Replacer replaces all occurrences from left to right
	// without overlapping and tries the old strings in argument order at
	// every position. Longer old strings first, so that the message of a
	// wrapped error which is also contained in the message of a sibling
	// wrapped error is not replaced within the sibling's occurrence.
	slices.SortStableFunc(replacements, func(a, b [2]string) int {
		return len(b[0]) - len(a[0])
	})
	oldNew := make([]string, 0, 2*len(replacements))
	for _, r := range replacements {
		oldNew = append(oldNew, r[0], r[1])
	}
	return strings.NewReplacer(oldNew...).Replace(msg)
}

// isNilPointer reports whether err is a non-nil interface
// holding a nil pointer, whose Error method usually panics.
func isNilPointer(err error) bool {
	v := reflect.ValueOf(err)
	return v.Kind() == reflect.Pointer && v.IsNil()
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
			return slices.ContainsFunc(x.Unwrap(), Type[T])
		default:
			return false
		}
	}
	return false
}
