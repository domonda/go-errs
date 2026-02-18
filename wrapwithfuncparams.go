package errs

/*
Call argument parameters are available on the stack,
but in a platform dependent packed format and not directly accessible
via package runtime.
Could only be parsed from runtime.Stack() result text.

See https://www.ardanlabs.com/blog/2015/01/stack-traces-in-go.html
*/

func wrapWithFuncParamsSkip(skip int, err error, params ...any) *withCallStackFuncParams {
	switch w := err.(type) {
	case callStackParamsProvider:
		// OK, wrap the wrapped
	case callStackProvider:
		// Already wrapped with call stack,
		// replace with withCallStackFuncParams
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

// WrapWithFuncParamsSkip wraps an error with the current call stack and function parameters,
// skipping skip stack frames.
//
// The skip parameter specifies how many stack frames to skip
// before capturing the call stack. Use skip=0 to capture the stack
// from the immediate caller of WrapWithFuncParamsSkip.
// Increase skip by 1 for each additional function wrapper you add.
//
// This function is typically used in defer statements to automatically
// wrap errors with the function's parameters when the function returns an error.
//
// Examples:
//
//	// Standard usage in defer - skip=0 is correct
//	func ProcessUser(userID string, age int) (err error) {
//	    defer WrapWithFuncParamsSkip(0, &err, userID, age)
//	    // ... function body
//	    return someError
//	}
//
//	// Wrapper function - skip=1+skip to pass through skip count
//	func myWrapperSkip(skip int, resultVar *error, params ...any) {
//	    WrapWithFuncParamsSkip(1+skip, resultVar, params...)
//	}
//
//	// Most common: use WrapWithFuncParams which has skip=0 built-in
//	func ProcessUser(userID string, age int) (err error) {
//	    defer WrapWithFuncParams(&err, userID, age)
//	    // ... function body
//	    return someError
//	}
func WrapWithFuncParamsSkip(skip int, resultVar *error, params ...any) {
	if *resultVar != nil {
		*resultVar = wrapWithFuncParamsSkip(1+skip, *resultVar, params...)
	}
}

// WrapWithFuncParams wraps an error with the current call stack and function parameters.
//
// This is the most commonly used function for wrapping errors with function parameters
// in defer statements. It automatically captures the correct call stack frame.
//
// Example:
//
//	func ProcessUser(userID string, age int) (err error) {
//	    defer WrapWithFuncParams(&err, userID, age)
//
//	    if age < 0 {
//	        return errors.New("invalid age")
//	    }
//	    // When an error is returned, it will be wrapped with:
//	    // - The call stack showing ProcessUser(userID, age)
//	    // - The actual parameter values passed to the function
//	    return someOperation(userID)
//	}
//
// Note: For functions with 0-10 parameters, use the optimized variants
// WrapWith0FuncParams through WrapWith10FuncParams for better performance.
func WrapWithFuncParams(resultVar *error, params ...any) {
	if *resultVar != nil {
		*resultVar = wrapWithFuncParamsSkip(1, *resultVar, params...)
	}
}

// WrapWith0FuncParams wraps an error with the call stack for functions with no parameters.
// This is more efficient than WrapWithFuncParams for zero-parameter functions.
func WrapWith0FuncParams(resultVar *error) {
	if *resultVar != nil {
		*resultVar = wrapWithFuncParamsSkip(1, *resultVar)
	}
}

// WrapWith1FuncParam wraps an error with the call stack and 1 function parameter.
// This is more efficient than WrapWithFuncParams for single-parameter functions.
func WrapWith1FuncParam(resultVar *error, p0 any) {
	if *resultVar != nil {
		*resultVar = wrapWithFuncParamsSkip(1, *resultVar, p0)
	}
}

// WrapWith2FuncParams wraps an error with the call stack and 2 function parameters.
// This is more efficient than WrapWithFuncParams for two-parameter functions.
func WrapWith2FuncParams(resultVar *error, p0, p1 any) {
	if *resultVar != nil {
		*resultVar = wrapWithFuncParamsSkip(1, *resultVar, p0, p1)
	}
}

// WrapWith3FuncParams wraps an error with the call stack and 3 function parameters.
// This is more efficient than WrapWithFuncParams for three-parameter functions.
func WrapWith3FuncParams(resultVar *error, p0, p1, p2 any) {
	if *resultVar != nil {
		*resultVar = wrapWithFuncParamsSkip(1, *resultVar, p0, p1, p2)
	}
}

// WrapWith4FuncParams wraps an error with the call stack and 4 function parameters.
// This is more efficient than WrapWithFuncParams for four-parameter functions.
func WrapWith4FuncParams(resultVar *error, p0, p1, p2, p3 any) {
	if *resultVar != nil {
		*resultVar = wrapWithFuncParamsSkip(1, *resultVar, p0, p1, p2, p3)
	}
}

// WrapWith5FuncParams wraps an error with the call stack and 5 function parameters.
// This is more efficient than WrapWithFuncParams for five-parameter functions.
func WrapWith5FuncParams(resultVar *error, p0, p1, p2, p3, p4 any) {
	if *resultVar != nil {
		*resultVar = wrapWithFuncParamsSkip(1, *resultVar, p0, p1, p2, p3, p4)
	}
}

// WrapWith6FuncParams wraps an error with the call stack and 6 function parameters.
// This is more efficient than WrapWithFuncParams for six-parameter functions.
func WrapWith6FuncParams(resultVar *error, p0, p1, p2, p3, p4, p5 any) {
	if *resultVar != nil {
		*resultVar = wrapWithFuncParamsSkip(1, *resultVar, p0, p1, p2, p3, p4, p5)
	}
}

// WrapWith7FuncParams wraps an error with the call stack and 7 function parameters.
// This is more efficient than WrapWithFuncParams for seven-parameter functions.
func WrapWith7FuncParams(resultVar *error, p0, p1, p2, p3, p4, p5, p6 any) {
	if *resultVar != nil {
		*resultVar = wrapWithFuncParamsSkip(1, *resultVar, p0, p1, p2, p3, p4, p5, p6)
	}
}

// WrapWith8FuncParams wraps an error with the call stack and 8 function parameters.
// This is more efficient than WrapWithFuncParams for eight-parameter functions.
func WrapWith8FuncParams(resultVar *error, p0, p1, p2, p3, p4, p5, p6, p7 any) {
	if *resultVar != nil {
		*resultVar = wrapWithFuncParamsSkip(1, *resultVar, p0, p1, p2, p3, p4, p5, p6, p7)
	}
}

// WrapWith9FuncParams wraps an error with the call stack and 9 function parameters.
// This is more efficient than WrapWithFuncParams for nine-parameter functions.
func WrapWith9FuncParams(resultVar *error, p0, p1, p2, p3, p4, p5, p6, p7, p8 any) {
	if *resultVar != nil {
		*resultVar = wrapWithFuncParamsSkip(1, *resultVar, p0, p1, p2, p3, p4, p5, p6, p7, p8)
	}
}

// WrapWith10FuncParams wraps an error with the call stack and 10 function parameters.
// This is more efficient than WrapWithFuncParams for ten-parameter functions.
func WrapWith10FuncParams(resultVar *error, p0, p1, p2, p3, p4, p5, p6, p7, p8, p9 any) {
	if *resultVar != nil {
		*resultVar = wrapWithFuncParamsSkip(1, *resultVar, p0, p1, p2, p3, p4, p5, p6, p7, p8, p9)
	}
}

type callStackParamsProvider interface {
	callStackProvider

	CallStackParams() ([]uintptr, []any)
}

var (
	_ error                   = &withCallStackFuncParams{}
	_ callStackProvider       = &withCallStackFuncParams{}
	_ callStackParamsProvider = &withCallStackFuncParams{}
)

// withCallStackFuncParams is an error wrapper that implements callStackParamsProvider
type withCallStackFuncParams struct {
	withCallStack

	params []any
}

func (w *withCallStackFuncParams) Error() string {
	return formatError(w)
}

func (w *withCallStackFuncParams) CallStackParams() ([]uintptr, []any) {
	return w.callStack, w.params
}
