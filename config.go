package errs

import (
	"runtime"
	"strings"

	"github.com/domonda/go-pretty"
)

// Configuration variables
var (
	// TrimFilePathPrefix will be trimmed from the
	// beginning of every call-stack file-path.
	// Defaults to $GOPATH/src/ of the build environment
	// or will be empty if go build gets called with -trimpath.
	TrimFilePathPrefix = filePathPrefix()

	// MaxCallStackFrames is the maximum number of frames to include in the call stack.
	MaxCallStackFrames = 32

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
	//	        return pretty.AsPrintable(v) // Use default
	//	    }
	//	}
	Printer = &pretty.DefaultPrinter

	// FormatParamMaxLen is the maximum length in bytes for a single formatted
	// parameter value in error call stacks. When a parameter's formatted
	// representation exceeds this length, it will be truncated and suffixed
	// with "…(TRUNCATED)".
	//
	// This prevents extremely large values (like long strings, big JSON blobs,
	// or large data structures) from making error messages unreadable.
	//
	// Default: 5000 bytes
	//
	// Example:
	//
	//	errs.FormatParamMaxLen = 100 // Limit to 100 bytes
	//
	//	func ProcessData(data string) (err error) {
	//	    defer errs.WrapWithFuncParams(&err, data)
	//	    // If data is 200 bytes, error will show:
	//	    // ProcessData("first 100 bytes of data…(TRUNCATED)")
	//	    return validateData(data)
	//	}
	FormatParamMaxLen = 5000
)

func filePathPrefix() string {
	// This Go package is hosted on GitHub
	// so there should always be "github.com"
	// in the path of this source file
	// if it was cloned using standard go get
	_, file, _, _ := runtime.Caller(1)
	end := strings.LastIndex(file, "github.com") // Works only if the source file is hosted on GitHub
	if end == -1 {
		// panic("expected github.com in call-stack file-path, but got: " + file)
		return "" // GitHub action might have it under /home/runner/work/...
	}
	return file[:end]
}
