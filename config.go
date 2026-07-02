package errs

import (
	"github.com/domonda/go-pretty"
)

// Configuration variables
var (
	// TrimFilePathPrefix will be trimmed from the
	// beginning of every call-stack file-path.
	//
	// It is empty by default, in which case call-stack file-paths are shown
	// in a checkout-independent import-path form (e.g.
	// github.com/domonda/go-errs/format.go), reconstructed from the frame's
	// package instead of the local filesystem path. This works in every
	// checkout, whether the module lives under $GOPATH/src, was built with
	// -trimpath, or sits in an arbitrary directory.
	//
	// Set it to a non-empty prefix to instead trim that exact prefix from the
	// raw runtime file-path (the pre-v1.0.4 behavior).
	TrimFilePathPrefix = ""

	// MaxCallStackFrames is the maximum number of frames to include in the call stack.
	MaxCallStackFrames = 32

	// Printer is the pretty.Printer used to format function parameters
	// in error call stacks. It can be configured to customize formatting,
	// mask secrets, or adapt types that don't implement pretty.Printable.
	//
	// Example - Masking sensitive data:
	//
	//	func init() {
	//	    errs.Printer = errs.Printer.WithPrintFuncFor(func(v reflect.Value) pretty.PrintFunc {
	//	        if v.Kind() == reflect.String && strings.Contains(v.String(), "secret") {
	//	            return func(w io.Writer) (int, error) {
	//	                return fmt.Fprint(w, "`***REDACTED***`")
	//	            }
	//	        }
	//	        return pretty.PrintFuncForPrintable(v) // Use default
	//	    })
	//	}
	Printer = &pretty.Printer{
		MaxStringLength: pretty.DefaultPrinter.MaxStringLength,
		MaxErrorLength:  pretty.DefaultPrinter.MaxErrorLength,
		MaxSliceLength:  pretty.DefaultPrinter.MaxSliceLength,
	}

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
