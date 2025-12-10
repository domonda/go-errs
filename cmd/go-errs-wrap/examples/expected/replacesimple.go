package examples

import (
	"context"

	"github.com/domonda/go-errs"
)

func ProcessDataSimple(ctx context.Context, id string, count int) (result string, err error) {
	defer errs.WrapWith3FuncParams(&err, ctx, id, count) // outdated params

	return "", nil
}

func LoadFileSimple(path string) (data []byte, err error) {
	defer errs.WrapWith1FuncParam(&err, path)

	return nil, nil
}

func NoParamsSimple() (err error) {
	defer errs.WrapWith0FuncParams(&err)
	return nil
}
