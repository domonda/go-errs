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
		"%s(%s)\n    %s:%d",
		frame.Function,
		formatParams(params),
		frame.File,
		frame.Line,
	)
}

func formatParams(params []interface{}) string {
	var b strings.Builder
	for i, param := range params {
		if i > 0 {
			b.WriteString(", ")
		}
		pretty.Fprint(&b, param)
	}
	return b.String()
}
