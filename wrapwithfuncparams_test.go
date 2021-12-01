package errs

import (
	"context"
	"fmt"
)

func funcA(ctx context.Context, i int, s string) (err error) {
	defer WrapWith3FuncParams(&err, ctx, i, s)

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
	err := funcA(context.Background(), 666, "Hello World!")
	fmt.Println(err)

	// Output:
	// error in funcC
	// github.com/domonda/go-errs.funcC()
	//     github.com/domonda/go-errs/wrapwithfuncparams_test.go:23
	// github.com/domonda/go-errs.funcB([`Hello World!`,"X\nX"])
	//     github.com/domonda/go-errs/wrapwithfuncparams_test.go:17
	// github.com/domonda/go-errs.funcA(Context{}, 666, `Hello World!`)
	//     github.com/domonda/go-errs/wrapwithfuncparams_test.go:11
}
