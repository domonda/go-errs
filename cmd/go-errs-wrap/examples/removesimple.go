package examples

import (
	"context"

	"github.com/domonda/go-errs"
)

func ProcessData(ctx context.Context, id string) (err error) {
	defer errs.WrapWithFuncParams(&err, ctx, id)

	return nil
}

func LoadFile(path string) (data []byte, err error) {
	//#wrap-result-err

	return nil, nil
}
