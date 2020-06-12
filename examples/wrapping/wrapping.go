package main

import (
	"errors"
	"fmt"

	"github.com/domonda/go-errs"
)

func funcA(i int, s string) (err error) {
	defer errs.WrapWithFuncParams(&err, i, s)

	return funcB(s)
}

func funcB(s ...string) (err error) {
	defer errs.WrapWithFuncParams(&err, s)

	return funcC()
}

func funcC() (err error) {
	defer errs.WrapWithFuncParams(&err)

	return errors.New("error in funcC")
}

func main() {
	err := funcA(666, "Hello World!")
	str := err.Error()
	fmt.Println(str)
}
