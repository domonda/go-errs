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
//
// MessageWithoutCallStacks removes the debug stack from the message.
func AsErrorWithDebugStack(val any) error {
	err := AsError(val)
	if err == nil {
		return nil
	}
	return &withDebugStack{
		err:     err,
		message: fmt.Sprintf("%v\n%s", err, debug.Stack()),
	}
}

// withDebugStack wraps an error with the text of debug.Stack()
// appended to its message.
type withDebugStack struct {
	err     error
	message string
}

func (e *withDebugStack) Error() string { return e.message }

func (e *withDebugStack) Unwrap() error { return e.err }

func (e *withDebugStack) messageWithoutCallStacks() string {
	return MessageWithoutCallStacks(e.err)
}

// panickedAfterError wraps the error of a recovered panic
// of a function that had already set its result error,
// which the message includes but which is not wrapped.
type panickedAfterError struct {
	result   error
	panicErr error
	message  string
}

func newPanickedAfterError(result, panicErr error) *panickedAfterError {
	return &panickedAfterError{
		result:   result,
		panicErr: panicErr,
		message:  panickedAfterErrorMessage(result, panicErr),
	}
}

// panickedAfterErrorMessage formats the message of a panickedAfterError
// from errors or from messages without call stacks.
func panickedAfterErrorMessage(result, panicErr any) string {
	return fmt.Sprintf("function returning error (%s) panicked with: %v", result, panicErr)
}

func (e *panickedAfterError) Error() string { return e.message }

func (e *panickedAfterError) Unwrap() error { return e.panicErr }

func (e *panickedAfterError) messageWithoutCallStacks() string {
	return panickedAfterErrorMessage(
		MessageWithoutCallStacks(e.result),
		MessageWithoutCallStacks(e.panicErr),
	)
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
		err = newPanickedAfterError(*result, err)
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
		err = newPanickedAfterError(*result, err)
	}

	*result = err
}
