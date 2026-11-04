package errs

import (
	"context"
	"fmt"
)

// The example has its own file because its output
// contains the source line numbers of funcA, funcB and funcC.

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
	//     github.com/domonda/go-errs/example_test.go:30
	// github.com/domonda/go-errs.funcB([`Hello World!`,`X\nX`])
	//     github.com/domonda/go-errs/example_test.go:24
	// github.com/domonda/go-errs.funcA(Context{}, 666, `Hello World!`, strct{A:-1})
	//     github.com/domonda/go-errs/example_test.go:18
}
