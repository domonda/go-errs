package errs

import (
	"errors"
	"fmt"
	"runtime/debug"
)

// AsError converts any type to an error without wrapping it.
// Nil values will be converted to a nil error.
func AsError(val any) error {
	switch x := val.(type) {
	case nil:
		return nil
	case error:
		return x
	case []error:
		return errors.Join(x...)
	case string:
		return errors.New(x)
	case fmt.Stringer:
		return errors.New(x.String())
	default:
		return errors.New(fmt.Sprint(val))
	}
}

// AsErrorWithDebugStack converts any type to an error
// and if not nil wraps it with debug.Stack() after a newline.
// Nil values will be converted to a nil error.
func AsErrorWithDebugStack(val any) error {
	err := AsError(val)
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w\n%s", err, debug.Stack())
}

// LogPanicWithFuncParams recovers any panic,
// converts it to an error wrapped with the callstack
// of the panic and the passed function parameter values
// and prints it with the prefix "LogPanicWithFuncParams: "
// to the passed Logger.
// After logging, the original panic is re-panicked.
func LogPanicWithFuncParams(log Logger, params ...any) {
	p := recover()
	if p == nil {
		return
	}

	err := AsErrorWithDebugStack(p)
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

	err := AsErrorWithDebugStack(p)
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

	err := AsErrorWithDebugStack(p)
	if *result != nil {
		err = fmt.Errorf("function returning error (%s) panicked with: %w", *result, err)
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

	err := AsErrorWithDebugStack(p)
	err = wrapWithFuncParamsSkip(1, err, params...)
	if *result != nil {
		err = fmt.Errorf("function returning error (%s) panicked with: %w", *result, err)
	}

	*result = err
}
