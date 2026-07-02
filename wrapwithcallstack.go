// Package errs provides Go 1.13+ compatible error wrapping with call stacks and function parameters.
//
// This package extends the standard library's error handling with:
//   - Automatic call stack capture for error context
//   - Function parameter tracking for detailed debugging
//   - Helper functions for common error patterns (NotFound, context errors)
//   - Panic recovery and conversion to errors
//   - Iterator support for Go 1.23+
//
// Basic usage:
//
//	func DoSomething(id string) (err error) {
//	    defer errs.WrapWithFuncParams(&err, id)
//	    // Your code here
//	    return someOperation(id)
//	}
//
// See the documentation of individual functions for more examples.
package errs

import (
	"fmt"
	"runtime"
	"slices"
)

// New returns a new error with the passed text
// wrapped with the current call stack.
func New(text string) error {
	return WrapWithCallStackSkip(1, Sentinel(text))
}

// Errorf wraps the result of fmt.Errorf with the current call stack.
//
// If the format specifier includes a %w verb with an error operand,
// the returned error will implement an Unwrap method returning the operand. It is
// invalid to include more than one %w verb or to supply it with an operand
// that does not implement the error interface. The %w verb is otherwise
// a synonym for %v.
func Errorf(format string, a ...any) error {
	return WrapWithCallStackSkip(1, fmt.Errorf(format, a...))
}

// WrapWithCallStack wraps an error with the current call stack.
func WrapWithCallStack(err error) error {
	return WrapWithCallStackSkip(1, err)
}

// WrapWithCallStackSkip wraps an error with the current call stack
// skipping skip stack frames.
//
// The skip parameter specifies how many stack frames to skip
// before capturing the call stack. Use skip=0 to capture the stack
// from the immediate caller of WrapWithCallStackSkip.
// Increase skip by 1 for each additional function wrapper you add.
//
// Examples:
//
//	// Direct use - skip=1 to show caller of your function
//	func DoSomething() error {
//	    err := someOperation()
//	    if err != nil {
//	        return WrapWithCallStackSkip(1, err)
//	    }
//	    return nil
//	}
//
//	// Wrapper function - skip=1+skip to pass through skip count
//	func myWrapper(skip int, err error) error {
//	    return WrapWithCallStackSkip(1+skip, err)
//	}
//
//	// Helper that wraps - skip=1 so caller of helper appears in stack
//	func wrapDatabaseError(err error) error {
//	    return WrapWithCallStackSkip(1, fmt.Errorf("database error: %w", err))
//	}
func WrapWithCallStackSkip(skip int, err error) error {
	if err == nil {
		return nil
	}
	return &withCallStack{
		err:       err,
		callStack: callStack(1 + skip),
	}
}

type callStackProvider interface {
	Unwrap() error
	CallStack() []uintptr
}

var (
	_ error                               = &withCallStack{}
	_ callStackProvider                   = &withCallStack{}
	_ interface{ StackTrace() []uintptr } = &withCallStack{}
)

// withCallStack is an error wrapper that implements callStackProvider
type withCallStack struct {
	err       error
	callStack []uintptr
}

func (w *withCallStack) Error() string {
	return formatError(w)
}

func (w *withCallStack) Unwrap() error {
	return w.err
}

func (w *withCallStack) CallStack() []uintptr {
	return w.callStack
}

// StackTrace returns the program counters of the captured call stack,
// ordered innermost first as returned by runtime.Callers.
//
// The method exists purely for interoperability with Sentry and other
// tools that discover an error's stack trace through reflection instead
// of a shared interface. sentry-go's ExtractStacktrace probes the error's
// concrete type for a method named StackFrames, StackTrace, or
// GetStackTracer (the pkg/errors, go-errors/errors, and pingcap/errors
// conventions) and reads program counters from the returned slice,
// accepting either a bare uintptr element or a struct with a
// ProgramCounter or PC field. Returning []uintptr under the pkg/errors
// StackTrace name satisfies that probe without importing any of those
// packages.
//
// Because withCallStackFuncParams embeds withCallStack, this method is
// promoted to it as well, so both wrapper types report a stack trace to
// Sentry.
//
// The returned slice is a copy, matching the pkg/errors contract, so a
// consumer that reorders it in place cannot corrupt the error's stored
// call stack.
func (w *withCallStack) StackTrace() []uintptr {
	return slices.Clone(w.callStack)
}

func callStack(skip int) []uintptr {
	c := make([]uintptr, MaxCallStackFrames)
	n := runtime.Callers(skip+2, c)
	return c[:n]
}
