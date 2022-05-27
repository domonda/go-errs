package errs

import (
	"context"
	"fmt"
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
	//     src/github.com/domonda/go-errs/wrapwithfuncparams_test.go:27
	// github.com/domonda/go-errs.funcB([`Hello World!`,`X\nX`])
	//     src/github.com/domonda/go-errs/wrapwithfuncparams_test.go:21
	// github.com/domonda/go-errs.funcA(Context{}, 666, `Hello World!`, strct{A:-1})
	//     src/github.com/domonda/go-errs/wrapwithfuncparams_test.go:15
}
