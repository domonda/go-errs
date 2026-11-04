package errs

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type strct struct {
	A int
}

func funcA(ctx context.Context, i int, s string, strct *strct) (err error) {
	defer WrapWith4FuncParams(&err, ctx, i, s, strct)

	return funcB(s, "X\nX")
}

func funcB(s ...string) (err error) {
	defer WrapWithFuncParams(&err, s)

	return funcC()
}

func funcC() (err error) {
	defer WrapWithFuncParams(&err)

	return New("error in funcC")
}

func ExampleWrapWithFuncParams() {
	err := funcA(context.Background(), 666, "Hello World!", &strct{A: -1})
	fmt.Println(err)

	// Output:
	// error in funcC
	// github.com/domonda/go-errs.funcC()
	//     github.com/domonda/go-errs/wrapwithfuncparams_test.go:33
	// github.com/domonda/go-errs.funcB([`Hello World!`,`X\nX`])
	//     github.com/domonda/go-errs/wrapwithfuncparams_test.go:27
	// github.com/domonda/go-errs.funcA(Context{}, 666, `Hello World!`, strct{A:-1})
	//     github.com/domonda/go-errs/wrapwithfuncparams_test.go:21
}

func TestWrapWithNFuncParams(t *testing.T) {
	tests := []struct {
		name       string
		wrap       func(resultVar *error)
		wantParams []any
	}{
		{name: "0", wrap: func(err *error) { WrapWith0FuncParams(err) }, wantParams: nil},
		{name: "1", wrap: func(err *error) { WrapWith1FuncParam(err, 0) }, wantParams: []any{0}},
		{name: "2", wrap: func(err *error) { WrapWith2FuncParams(err, 0, 1) }, wantParams: []any{0, 1}},
		{name: "3", wrap: func(err *error) { WrapWith3FuncParams(err, 0, 1, 2) }, wantParams: []any{0, 1, 2}},
		{name: "4", wrap: func(err *error) { WrapWith4FuncParams(err, 0, 1, 2, 3) }, wantParams: []any{0, 1, 2, 3}},
		{name: "5", wrap: func(err *error) { WrapWith5FuncParams(err, 0, 1, 2, 3, 4) }, wantParams: []any{0, 1, 2, 3, 4}},
		{name: "6", wrap: func(err *error) { WrapWith6FuncParams(err, 0, 1, 2, 3, 4, 5) }, wantParams: []any{0, 1, 2, 3, 4, 5}},
		{name: "7", wrap: func(err *error) { WrapWith7FuncParams(err, 0, 1, 2, 3, 4, 5, 6) }, wantParams: []any{0, 1, 2, 3, 4, 5, 6}},
		{name: "8", wrap: func(err *error) { WrapWith8FuncParams(err, 0, 1, 2, 3, 4, 5, 6, 7) }, wantParams: []any{0, 1, 2, 3, 4, 5, 6, 7}},
		{name: "9", wrap: func(err *error) { WrapWith9FuncParams(err, 0, 1, 2, 3, 4, 5, 6, 7, 8) }, wantParams: []any{0, 1, 2, 3, 4, 5, 6, 7, 8}},
		{name: "10", wrap: func(err *error) { WrapWith10FuncParams(err, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9) }, wantParams: []any{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}},
		{name: "variadic", wrap: func(err *error) { WrapWithFuncParams(err, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10) }, wantParams: []any{0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var err error
			tt.wrap(&err)
			assert.NoError(t, err, "nil error must stay nil")

			sentinel := errors.New("sentinel")
			err = sentinel
			tt.wrap(&err)
			assert.ErrorIs(t, err, sentinel)
			wrapped, ok := errors.AsType[*withCallStackFuncParams](err)
			require.True(t, ok)
			assert.Equal(t, tt.wantParams, wrapped.params)
		})
	}
}

// wrapResultInHelper wraps through a helper function
// and skips its own frame to record the call stack of its caller.
func wrapResultInHelper(resultVar *error, params ...any) {
	WrapWithFuncParamsSkip(1, resultVar, params...)
}

func returnErrorWrappedInHelper(err error) (resultErr error) {
	defer wrapResultInHelper(&resultErr, "param")
	return err
}

func TestWrapWithFuncParamsSkip(t *testing.T) {
	assert.NoError(t, returnErrorWrappedInHelper(nil), "nil error must stay nil")

	sentinel := errors.New("sentinel")
	err := returnErrorWrappedInHelper(sentinel)
	assert.ErrorIs(t, err, sentinel)
	wrapped, ok := errors.AsType[*withCallStackFuncParams](err)
	require.True(t, ok)
	assert.Equal(t, []any{"param"}, wrapped.params)
	frame, _ := runtime.CallersFrames(wrapped.callStack).Next()
	assert.Equal(t, "github.com/domonda/go-errs.returnErrorWrappedInHelper", frame.Function)
}
