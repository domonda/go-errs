package errs

import (
	"errors"
	"fmt"
	"runtime"
	"strings"

	"github.com/domonda/go-pretty"
)

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
		firstWithoutStack = errors.New("wraperr: no wrapped error found")
	}

	var b strings.Builder
	fmt.Fprintln(&b, firstWithoutStack.Error())
	for i := len(calls) - 1; i >= 0; i-- {
		fmt.Fprintln(&b, calls[i])
	}
	return b.String()
}

func formatCallStack(e callStackProvider) string {
	stack := e.CallStack()
	frame, ok := runtime.CallersFrames(stack).Next()
	if !ok {
		return "insufficient call stack"
	}
	return fmt.Sprintf(
		"%s\n    %s:%d",
		frame.Function,
		frame.File,
		frame.Line,
	)
}

func formatCallStackParams(e callStackParamsProvider) string {
	stack, params := e.CallStackParams()
	frame, ok := runtime.CallersFrames(stack).Next()
	if !ok {
		return "insufficient call stack"
	}
	return fmt.Sprintf(
		"%s\n    %s:%d",
		FormatFunctionCall(frame.Function, params...),
		frame.File,
		frame.Line,
	)
}

// FormatFunctionCall formats a function call in pseudo syntax
// using github.com/domonda/go-pretty to format the params.
// Used to format errors with function call stack information.
func FormatFunctionCall(function string, params ...interface{}) string {
	var b strings.Builder
	b.WriteString(function)
	b.WriteByte('(')
	for i, param := range params {
		if i > 0 {
			b.WriteString(", ")
		}
		pretty.Fprint(&b, param)
	}
	b.WriteByte(')')
	return b.String()
}
