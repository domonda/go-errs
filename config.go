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
