package errs

import (
	"errors"
	"fmt"
	"runtime/debug"

	"github.com/domonda/go-pretty"
)

// AsError converts val to an error without wrapping it.
func AsError(val interface{}) error {
	switch x := val.(type) {
	case nil:
		return nil
	case error:
		return x
	case string:
		return errors.New(x)
	case fmt.Stringer:
		return errors.New(x.String())
	default:
		return errors.New(pretty.Sprint(val))
	}
}

func LogPanicWithFuncParams(log Logger, params ...interface{}) {
	p := recover()
	if p == nil {
		return
	}

	err := fmt.Errorf("%w\n%s", AsError(p), debug.Stack())
	err = wrapWithFuncParamsSkip(1, err, params...)

	log.Printf("LogPanicWithFuncParams: %s", err.Error())

	panic(p)
}

func RecoverAndLogPanicWithFuncParams(log Logger, params ...interface{}) {
	p := recover()
	if p == nil {
		return
	}

	err := fmt.Errorf("%w\n%s", AsError(p), debug.Stack())
	err = wrapWithFuncParamsSkip(1, err, params...)

	log.Printf("RecoverAndLogPanicWithFuncParams: %s", err.Error())
}

func RecoverPanicAsErrorWithFuncParams(resultVar *error, params ...interface{}) {
	p := recover()
	if p == nil {
		return
	}

	err := fmt.Errorf("%w\n%s", AsError(p), debug.Stack())
	err = wrapWithFuncParamsSkip(1, err, params...)
	if *resultVar != nil {
		err = fmt.Errorf("function returning error (%s) paniced with: %w", *resultVar, err)
	}

	*resultVar = err
}
