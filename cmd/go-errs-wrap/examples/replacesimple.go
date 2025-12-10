package examples

import (
	"context"

	"github.com/domonda/go-errs"
)

func ProcessDataSimple(ctx context.Context, id string, count int) (result string, err error) {
	defer errs.WrapWithFuncParams(&err, ctx, id) // outdated params

	return "", nil
}

func LoadFileSimple(path string) (data []byte, err error) {
	//#wrap-result-err

	return nil, nil
}

func NoParamsSimple() (err error) {
	//#wrap-result-err
	return nil
}
