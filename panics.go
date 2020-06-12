package errs

import (
	"fmt"

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
		return Sentinel(x)
	case fmt.Stringer:
		return Sentinel(x.String())
	default:
		return Sentinel(pretty.Sprint(val))
	}
}

func LogPanicWithFuncParams(log Logger, params ...interface{}) {
	p := recover()
	if p == nil {
		return
	}

	err := wrapWithFuncParamsSkip(1, AsError(p), params...)

	log.Printf("LogPanicWithFuncParams: %s", err.Error())

	panic(p)
}

func RecoverAndLogPanicWithFuncParams(log Logger, params ...interface{}) {
	p := recover()
	if p == nil {
		return
	}

	err := wrapWithFuncParamsSkip(1, AsError(p), params...)

	log.Printf("RecoverAndLogPanicWithFuncParams: %s", err.Error())
}

func RecoverPanicAsErrorWithFuncParams(resultVar *error, params ...interface{}) {
	p := recover()
	if p == nil {
		return
	}

	err := wrapWithFuncParamsSkip(1, AsError(p), params...)

	if *resultVar != nil {
		*resultVar = fmt.Errorf("function returning error %s paniced with: %w", *resultVar, err)
	} else {
		*resultVar = err
	}
}
