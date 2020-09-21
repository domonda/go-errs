package errs

/*
Call argument parameters are available on the stack,
but in a platform dependent packed format and not directly accessible
via package runtime.
Could only be parsed from runtime.Stack() result text.

See https://www.ardanlabs.com/blog/2015/01/stack-traces-in-go.html
*/

func wrapWithFuncParamsSkip(skip int, err error, params ...interface{}) *withCallStackFuncParams {
	switch w := err.(type) {
	case callStackParamsProvider:
		// OK, wrap the wrapped
	case callStackProvider:
		// Already wrapped with stack,
		// replace wrapper wrapWithStackParams
		return &withCallStackFuncParams{
			withCallStack: withCallStack{
				err:       w.Unwrap(),
				callStack: w.CallStack(),
			},
			params: params,
		}
	}

	return &withCallStackFuncParams{
		withCallStack: withCallStack{
			err:       err,
			callStack: callStack(skip + 1),
		},
		params: params,
	}
}

func WrapWithFuncParamsSkip(skip int, resultVar *error, params ...interface{}) {
	if *resultVar != nil {
		*resultVar = wrapWithFuncParamsSkip(1+skip, *resultVar, params...)
	}
}

func WrapWithFuncParams(resultVar *error, params ...interface{}) {
	if *resultVar != nil {
		*resultVar = wrapWithFuncParamsSkip(1, *resultVar, params...)
	}
}

func WrapWith0FuncParams(resultVar *error) {
	if *resultVar != nil {
		*resultVar = wrapWithFuncParamsSkip(1, *resultVar)
	}
}

func WrapWith1FuncParams(resultVar *error, p0 interface{}) {
	if *resultVar != nil {
		*resultVar = wrapWithFuncParamsSkip(1, *resultVar, p0)
	}
}

func WrapWith2FuncParams(resultVar *error, p0, p1 interface{}) {
	if *resultVar != nil {
		*resultVar = wrapWithFuncParamsSkip(1, *resultVar, p0, p1)
	}
}

func WrapWith3FuncParams(resultVar *error, p0, p1, p2 interface{}) {
	if *resultVar != nil {
		*resultVar = wrapWithFuncParamsSkip(1, *resultVar, p0, p1, p2)
	}
}

func WrapWith4FuncParams(resultVar *error, p0, p1, p2, p3 interface{}) {
	if *resultVar != nil {
		*resultVar = wrapWithFuncParamsSkip(1, *resultVar, p0, p1, p2, p3)
	}
}

func WrapWith5FuncParams(resultVar *error, p0, p1, p2, p3, p4 interface{}) {
	if *resultVar != nil {
		*resultVar = wrapWithFuncParamsSkip(1, *resultVar, p0, p1, p2, p3, p4)
	}
}

func WrapWith6FuncParams(resultVar *error, p0, p1, p2, p3, p4, p5 interface{}) {
	if *resultVar != nil {
		*resultVar = wrapWithFuncParamsSkip(1, *resultVar, p0, p1, p2, p3, p4, p5)
	}
}

func WrapWith7FuncParams(resultVar *error, p0, p1, p2, p3, p4, p5, p6 interface{}) {
	if *resultVar != nil {
		*resultVar = wrapWithFuncParamsSkip(1, *resultVar, p0, p1, p2, p3, p4, p5, p6)
	}
}

func WrapWith8FuncParams(resultVar *error, p0, p1, p2, p3, p4, p5, p6, p7 interface{}) {
	if *resultVar != nil {
		*resultVar = wrapWithFuncParamsSkip(1, *resultVar, p0, p1, p2, p3, p4, p5, p6, p7)
	}
}

func WrapWith9FuncParams(resultVar *error, p0, p1, p2, p3, p4, p5, p6, p7, p8 interface{}) {
	if *resultVar != nil {
		*resultVar = wrapWithFuncParamsSkip(1, *resultVar, p0, p1, p2, p3, p4, p5, p6, p7, p8)
	}
}

func WrapWith10FuncParams(resultVar *error, p0, p1, p2, p3, p4, p5, p6, p7, p8, p9 interface{}) {
	if *resultVar != nil {
		*resultVar = wrapWithFuncParamsSkip(1, *resultVar, p0, p1, p2, p3, p4, p5, p6, p7, p8, p9)
	}
}

type callStackParamsProvider interface {
	callStackProvider

	CallStackParams() ([]uintptr, []interface{})
}

type withCallStackFuncParams struct {
	withCallStack

	params []interface{}
}

func (w *withCallStackFuncParams) Error() string {
	return formatError(w)
}

func (w *withCallStackFuncParams) CallStackParams() ([]uintptr, []interface{}) {
	return w.callStack, w.params
}
