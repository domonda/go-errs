package examples

import "github.com/domonda/go-errs"

func Outer(a int) (err error) {
	inner := func(b string) (innerErr error) {
		defer errs.WrapWith1FuncParam(&innerErr, b)
		return nil
	}
	_ = inner
	return nil
}
