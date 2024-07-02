package errs

import (
	"errors"
	"fmt"
	"io"
	"runtime"
	"strings"

	"github.com/domonda/go-pretty"
)

// CallStackPrintable can be implemented to customize the printing
// of the implementation's data in an error call stack output.
type CallStackPrintable interface {
	PrintForCallStack(io.Writer)
}

func formatError(err error) string {
	var (
		firstWithoutStack error
		calls             []string
	)

	for err != nil {
		switch e := err.(type) {
		case callStackParamsProvider:
			calls = append(calls, formatCallStackParams(e))

		case callStackProvider:
			calls = append(calls, formatCallStack(e))

		default:
			if firstWithoutStack == nil {
				firstWithoutStack = err
			}
		}

		err = errors.Unwrap(err)
	}

	if firstWithoutStack == nil {
		// Should never happen, just to make sure we don't panic
		firstWithoutStack = errors.New("no wrapped error found")
	}

	var b strings.Builder
	b.WriteString(firstWithoutStack.Error()) //#nosec
	b.WriteByte('\n')                        //#nosec
	for i := len(calls) - 1; i >= 0; i-- {
		b.WriteString(calls[i]) //#nosec
		b.WriteByte('\n')       //#nosec
	}
	return b.String()
}

func formatCallStack(e callStackProvider) string {
	stack := e.CallStack()
	frame, _ := runtime.CallersFrames(stack).Next()
	return fmt.Sprintf(
		"%s\n    %s:%d",
		frame.Function,
		strings.TrimPrefix(frame.File, TrimFilePathPrefix),
		frame.Line,
	)
}

func formatCallStackParams(e callStackParamsProvider) string {
	stack, params := e.CallStackParams()
	frame, _ := runtime.CallersFrames(stack).Next()
	return fmt.Sprintf(
		"%s\n    %s:%d",
		FormatFunctionCall(frame.Function, params...),
		strings.TrimPrefix(frame.File, TrimFilePathPrefix),
		frame.Line,
	)
}

// FormatFunctionCall formats a function call in pseudo syntax
// using the PrintForCallStack method of params that implement
// the CallStackPrintable interface or github.com/domonda/go-pretty
// to format params that don't implement CallStackPrintable.
//
// FormatFunctionCall is a function variable that can be changed
// to globally configure the formatting of function calls.
var FormatFunctionCall = func(function string, params ...any) string {
	var b strings.Builder
	b.WriteString(function)
	b.WriteByte('(')
	for i, param := range params {
		if i > 0 {
			b.WriteString(", ")
		}
		if printable, ok := param.(CallStackPrintable); ok {
			printable.PrintForCallStack(&b)
		} else {
			pretty.Fprint(&b, param)
		}
	}
	b.WriteByte(')')
	return b.String()
}

// LogFunctionCall using FormatFunctionCall if logger is not nil
func LogFunctionCall(logger Logger, function string, params ...any) {
	if logger != nil {
		logger.Printf(FormatFunctionCall(function, params...))
	}
}
