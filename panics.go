package errs

import (
	"errors"
	"fmt"
	"runtime/debug"

	"github.com/domonda/go-pretty"
)

// AsError converts any type to an error without wrapping it.
func AsError(val any) error {
	switch x := val.(type) {
	case nil:
		return nil
	case error:
		return x
	case []error:
		return Combine(x...)
	case string:
		return errors.New(x)
	case fmt.Stringer:
		return errors.New(x.String())
	default:
		return errors.New(pretty.Sprint(val))
	}
}

// LogPanicWithFuncParams recovers any panic,
// converts it to an error wrapped with the callstack
// of the panic and the passed function parameter values
// and prints it with the prefix "LogPanicWithFuncParams: "
// to the passed Logger.
// After logging, the original panic is re-paniced.
func LogPanicWithFuncParams(log Logger, params ...any) {
	p := recover()
	if p == nil {
		return
	}

	err := fmt.Errorf("%w\n%s", AsError(p), debug.Stack())
	err = wrapWithFuncParamsSkip(1, err, params...)

	log.Printf("LogPanicWithFuncParams: %s", err.Error())

	panic(p)
}

// RecoverAndLogPanicWithFuncParams recovers any panic,
// converts it to an error wrapped with the callstack
// of the panic and the passed function parameter values
// and prints it with the prefix "RecoverAndLogPanicWithFuncParams: "
// to the passed Logger.
func RecoverAndLogPanicWithFuncParams(log Logger, params ...any) {
	p := recover()
	if p == nil {
		return
	}

	err := fmt.Errorf("%w\n%s", AsError(p), debug.Stack())
	err = wrapWithFuncParamsSkip(1, err, params...)

	log.Printf("RecoverAndLogPanicWithFuncParams: %s", err.Error())
}

// RecoverPanicAsError recovers any panic,
// converts it to an error wrapped with the callstack
// of the panic and assigns it to the result error.
func RecoverPanicAsError(result *error) {
	p := recover()
	if p == nil {
		return
	}

	err := fmt.Errorf("%w\n%s", AsError(p), debug.Stack())
	if *result != nil {
		err = fmt.Errorf("function returning error (%s) paniced with: %w", *result, err)
	}

	*result = err
}

// RecoverPanicAsErrorWithFuncParams recovers any panic,
// converts it to an error wrapped with the callstack
// of the panic and the passed function parameter values
// and assigns it to the result error.
func RecoverPanicAsErrorWithFuncParams(result *error, params ...any) {
	p := recover()
	if p == nil {
		return
	}

	err := fmt.Errorf("%w\n%s", AsError(p), debug.Stack())
	err = wrapWithFuncParamsSkip(1, err, params...)
	if *result != nil {
		err = fmt.Errorf("function returning error (%s) paniced with: %w", *result, err)
	}

	*result = err
}
