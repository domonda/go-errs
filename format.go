package errs

import (
	"bytes"
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
)

// FormatFunctionCall formats a function call with parameters using go-pretty.
//
// Types can implement pretty.Stringer, pretty.Printable, or pretty.PrintableWithResult
// from github.com/domonda/go-pretty to customize their representation in error call
// stacks and other formatted output.
//
// Example:
//
//	type SensitiveData struct {
//	    value string
//	}
//
//	func (SensitiveData) PrettyString() string {
//	    return "***REDACTED***"
//	}
//
// Since go-pretty handles recursive checking in nested struct fields,
// types are properly formatted at any nesting level.

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
		callStackFilePath(frame),
		frame.Line,
	)
}

func formatCallStackParams(e callStackParamsProvider) string {
	stack, params := e.CallStackParams()
	frame, _ := runtime.CallersFrames(stack).Next()
	return fmt.Sprintf(
		"%s\n    %s:%d",
		FormatFunctionCall(frame.Function, params...),
		callStackFilePath(frame),
		frame.Line,
	)
}

// callStackFilePath returns the source file path to display for a stack frame.
//
// If [TrimFilePathPrefix] is set it is trimmed from the raw runtime file-path
// (legacy behavior). Otherwise the path is returned in a checkout-independent
// import-path form: a file-path that is already relative (built with -trimpath)
// is used as-is, while an absolute build path is reconstructed from the frame's
// package import path (always carried by frame.Function) and the file's base
// name. That makes the output identical no matter where the module is checked
// out.
func callStackFilePath(frame runtime.Frame) string {
	file := frame.File
	if TrimFilePathPrefix != "" {
		return strings.TrimPrefix(file, TrimFilePathPrefix)
	}
	if file == "" || !filepath.IsAbs(file) {
		return file
	}
	pkg := funcPackagePath(frame.Function)
	if pkg == "" {
		return file
	}
	return pkg + "/" + filepath.Base(file)
}

// funcPackagePath extracts the package import path from a fully-qualified
// function name as reported by [runtime.Frame.Function], for example
// "github.com/domonda/go-errs.funcC"        -> "github.com/domonda/go-errs"
// "github.com/domonda/go-errs.(*T).Method"  -> "github.com/domonda/go-errs"
// "github.com/domonda/go-errs.New.func1"    -> "github.com/domonda/go-errs"
//
// The package path is everything before the first '.' that follows the last
// '/', because the final import-path element (the package directory name) never
// contains a '.'. Returns "" if fn carries no package path.
func funcPackagePath(fn string) string {
	lastSlash := strings.LastIndexByte(fn, '/')
	dot := strings.IndexByte(fn[lastSlash+1:], '.')
	if dot < 0 {
		return ""
	}
	return fn[:lastSlash+1+dot]
}

// FormatFunctionCall formats a function call in pseudo syntax
// using the Printer variable to format parameters.
// Types implementing pretty.PrintableWithResult, pretty.Printable,
// or pretty.Stringer will use their respective methods,
// and this works recursively for nested structs.
//
// FormatFunctionCall is a function variable that can be changed
// to globally configure the formatting of function calls.
//
// Default Implementation:
//
// The default implementation formats function calls as:
//
//	functionName(param1, param2, ...)
//
// Each parameter is formatted using the Printer variable. If a formatted
// parameter exceeds FormatParamMaxLen bytes, it will be truncated to ensure
// valid UTF-8 and suffixed with "…(TRUNCATED)".
var FormatFunctionCall = func(function string, params ...any) string {
	var b strings.Builder
	b.WriteString(function)
	b.WriteByte('(')
	for i, param := range params {
		if i > 0 {
			b.WriteString(", ")
		}
		paramStr := Printer.Sprint(param)
		if len(paramStr) > FormatParamMaxLen {
			// Cut off slice may end with invalid UTF-8 sequence
			b.Write(bytes.ToValidUTF8([]byte(paramStr[:FormatParamMaxLen]), nil))
			b.WriteString("…(TRUNCATED)")
		} else {
			b.WriteString(paramStr)
		}
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
