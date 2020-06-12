package errs

import (
	"context"
	"fmt"
	"runtime"
	"strings"
)

func funcA(ctx context.Context, i int, s string) (err error) {
	defer WrapWithFuncParams(&err, ctx, i, s)

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

func basePath() string {
	_, file, _, _ := runtime.Caller(1)
	return file[:strings.Index(file, "github.com")]
}

func ExampleWrapWithFuncParams() {
	err := funcA(context.Background(), 666, "Hello World!")
	errStr := err.Error()
	errStr = strings.ReplaceAll(errStr, basePath(), "")
	fmt.Println(errStr)

	// Output:
	// error in funcC
	// github.com/domonda/go-errs.funcC()
	//     github.com/domonda/go-errs/wrapwithfuncparams_test.go:25
	// github.com/domonda/go-errs.funcB([]string{"Hello World!", "X\nX"})
	//     github.com/domonda/go-errs/wrapwithfuncparams_test.go:19
	// github.com/domonda/go-errs.funcA(Context{Err:<nil>}, 666, "Hello World!")
	//     github.com/domonda/go-errs/wrapwithfuncparams_test.go:13
}
