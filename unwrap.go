package errs

import (
	"errors"
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
// of the error tree. It returns "" for a nil err.
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
// removes the outermost call-stack wrappers, then replaces the Error result
// of each wrapped error (from Unwrap() error or Unwrap() []error) in the
// message with the wrapped error's message without call stacks,
// recursively. This handles fmt.Errorf with one or more %w verbs and
// errors.Join.
//
// A wrapper whose Error method does not contain the Error result of
// its wrapped error verbatim keeps the message it rendered,
// including any call stack text it copied in a changed form.
// The same applies when a wrapped error's message changed after the wrapper
// rendered it, for example because a function parameter passed to
// WrapWithFuncParams is a pointer to a value that was modified later.
func MessageWithoutCallStacks(err error) string {
	err = UnwrapCallStack(err)
	if err == nil {
		return ""
	}
	var wrapped []error
	switch x := err.(type) {
	case interface{ Unwrap() error }:
		wrapped = []error{x.Unwrap()}
	case interface{ Unwrap() []error }:
		wrapped = x.Unwrap()
	}
	type wrappedMessage struct{ withStacks, withoutStacks string }
	var messages []wrappedMessage
	for _, w := range wrapped {
		if w == nil {
			continue
		}
		m := wrappedMessage{w.Error(), MessageWithoutCallStacks(w)}
		if m.withStacks != m.withoutStacks {
			messages = append(messages, m)
		}
	}
	// Longer messages first, so that the message of a wrapped error which is
	// also contained in the message of a sibling wrapped error
	// is not replaced within the sibling's occurrence.
	slices.SortStableFunc(messages, func(a, b wrappedMessage) int {
		return len(b.withStacks) - len(a.withStacks)
	})
	// Occurrences are searched in the original message and may not overlap,
	// so wrapped errors with identical messages each replace their own one.
	msg := err.Error()
	var replacements []messageReplacement
	for _, m := range messages {
		start := indexNotOverlapping(msg, m.withStacks, replacements)
		if start >= 0 {
			replacements = append(replacements, messageReplacement{start, start + len(m.withStacks), m.withoutStacks})
		}
	}
	if len(replacements) == 0 {
		return msg
	}
	slices.SortFunc(replacements, func(a, b messageReplacement) int { return a.start - b.start })
	var b strings.Builder
	pos := 0
	for _, r := range replacements {
		b.WriteString(msg[pos:r.start])
		b.WriteString(r.text)
		pos = r.end
	}
	b.WriteString(msg[pos:])
	return b.String()
}

// messageReplacement replaces msg[start:end] with text.
type messageReplacement struct {
	start, end int
	text       string
}

// indexNotOverlapping returns the index of the first occurrence of substr in s
// that overlaps none of the replacements, or -1 if there is none.
func indexNotOverlapping(s, substr string, replacements []messageReplacement) int {
	from := 0
search:
	for from <= len(s)-len(substr) {
		i := strings.Index(s[from:], substr)
		if i < 0 {
			return -1
		}
		start, end := from+i, from+i+len(substr)
		for _, r := range replacements {
			if start < r.end && r.start < end {
				from = r.end
				continue search
			}
		}
		return start
	}
	return -1
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
