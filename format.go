package errs

import (
	"errors"
	"fmt"
	"runtime"
	"strings"

	"github.com/domonda/go-pretty"
)

// Printer is the pretty.Printer used to format function parameters
// in error call stacks. It can be configured to customize formatting,
// mask secrets, or adapt types that don't implement pretty.Printable.
//
// Example - Masking sensitive data:
//
//	func init() {
//	    errs.Printer.AsPrintable = func(v reflect.Value) (pretty.Printable, bool) {
//	        if v.Kind() == reflect.String && strings.Contains(v.String(), "secret") {
//	            return printableAdapter{
//	                format: func(w io.Writer) {
//	                    fmt.Fprint(w, "`***REDACTED***`")
//	                },
//	            }, true
//	        }
//	        return nil, false
//	    }
//	}
var Printer = &pretty.DefaultPrinter

// FormatFunctionCall formats a function call with parameters using pretty.Printable.
//
// Types can implement pretty.Printable from github.com/domonda/go-pretty to customize
// their representation in error call stacks and other formatted output.
//
// Example:
//
//	type SensitiveData struct {
//	    value string
//	}
//
//	func (s SensitiveData) PrettyPrint(w io.Writer) {
//	    io.WriteString(w, "***REDACTED***")
//	}
//
// Since go-pretty already handles recursive checking of pretty.Printable implementations
// in nested struct fields, types are properly formatted at any nesting level.

// formatError formats an error with its call stack and function parameters.
// It unwraps the error chain and builds a formatted string showing:
//   - The root error message
//   - Each function call with its parameters (if wrapped with WrapWithFuncParams)
//   - The file and line number for each call
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
// using the Printer variable to format parameters. Types that implement pretty.Printable
// will use their PrettyPrint method, and this works recursively for nested structs.
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
		Printer.Fprint(&b, param)
	}
	b.WriteByte(')')
	return b.String()
}

// LogFunctionCall logs a formatted function call using FormatFunctionCall if logger is not nil.
// This is useful for logging function calls with their parameters for debugging.
func LogFunctionCall(logger Logger, function string, params ...any) {
	if logger != nil {
		logger.Printf(FormatFunctionCall(function, params...))
	}
}
