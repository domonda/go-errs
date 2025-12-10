package examples

import "github.com/domonda/go-errs"

func ManyParams(a, b, c, d, e, f, g, h, i, j, k string) (err error) {
	defer errs.WrapWithFuncParams(&err, a, b, c, d, e, f, g, h, i, j, k)
	return nil
}

func TenParams(p0, p1, p2, p3, p4, p5, p6, p7, p8, p9 int) (err error) {
	defer errs.WrapWith10FuncParams(&err, p0, p1, p2, p3, p4, p5, p6, p7, p8, p9)
	return nil
}
