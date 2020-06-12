package errs

import (
	"fmt"
	"runtime"
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
func Errorf(format string, a ...interface{}) error {
	return WrapWithCallStackSkip(1, fmt.Errorf(format, a...))
}

// WrapWithCallStack wraps an error with the current call stack.
func WrapWithCallStack(err error) error {
	return WrapWithCallStackSkip(1, err)
}

// WrapWithCallStackSkip wraps an error with the current call stack
// skipping skip stack frames.
func WrapWithCallStackSkip(skip int, err error) error {
	return &withCallStack{
		err:       err,
		callStack: callStack(1 + skip),
	}
}

type callStackProvider interface {
	Unwrap() error
	CallStack() []uintptr
}

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

func callStack(skip int) []uintptr {
	c := make([]uintptr, 32)
	n := runtime.Callers(skip+2, c)
	return c[:n]
}
