package rewrite

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateWrapStatement(t *testing.T) {
	tests := []struct {
		name       string
		ctx        *funcContext
		wantResult string
	}{
		{
			name: "0 parameters",
			ctx: &funcContext{
				funcName:        "NoParams",
				paramNames:      nil,
				errorResultName: "err",
			},
			wantResult: "defer errs.WrapWith0FuncParams(&err)",
		},
		{
			name: "1 parameter",
			ctx: &funcContext{
				funcName:        "OneParam",
				paramNames:      []string{"ctx"},
				errorResultName: "err",
			},
			wantResult: "defer errs.WrapWith1FuncParam(&err, ctx)",
		},
		{
			name: "2 parameters",
			ctx: &funcContext{
				funcName:        "TwoParams",
				paramNames:      []string{"ctx", "id"},
				errorResultName: "err",
			},
			wantResult: "defer errs.WrapWith2FuncParams(&err, ctx, id)",
		},
		{
			name: "3 parameters",
			ctx: &funcContext{
				funcName:        "ThreeParams",
				paramNames:      []string{"ctx", "id", "name"},
				errorResultName: "err",
			},
			wantResult: "defer errs.WrapWith3FuncParams(&err, ctx, id, name)",
		},
		{
			name: "4 parameters",
			ctx: &funcContext{
				funcName:        "FourParams",
				paramNames:      []string{"a", "b", "c", "d"},
				errorResultName: "err",
			},
			wantResult: "defer errs.WrapWith4FuncParams(&err, a, b, c, d)",
		},
		{
			name: "5 parameters",
			ctx: &funcContext{
				funcName:        "FiveParams",
				paramNames:      []string{"a", "b", "c", "d", "e"},
				errorResultName: "err",
			},
			wantResult: "defer errs.WrapWith5FuncParams(&err, a, b, c, d, e)",
		},
		{
			name: "6 parameters",
			ctx: &funcContext{
				funcName:        "SixParams",
				paramNames:      []string{"a", "b", "c", "d", "e", "f"},
				errorResultName: "err",
			},
			wantResult: "defer errs.WrapWith6FuncParams(&err, a, b, c, d, e, f)",
		},
		{
			name: "7 parameters",
			ctx: &funcContext{
				funcName:        "SevenParams",
				paramNames:      []string{"a", "b", "c", "d", "e", "f", "g"},
				errorResultName: "err",
			},
			wantResult: "defer errs.WrapWith7FuncParams(&err, a, b, c, d, e, f, g)",
		},
		{
			name: "8 parameters",
			ctx: &funcContext{
				funcName:        "EightParams",
				paramNames:      []string{"a", "b", "c", "d", "e", "f", "g", "h"},
				errorResultName: "err",
			},
			wantResult: "defer errs.WrapWith8FuncParams(&err, a, b, c, d, e, f, g, h)",
		},
		{
			name: "9 parameters",
			ctx: &funcContext{
				funcName:        "NineParams",
				paramNames:      []string{"a", "b", "c", "d", "e", "f", "g", "h", "i"},
				errorResultName: "err",
			},
			wantResult: "defer errs.WrapWith9FuncParams(&err, a, b, c, d, e, f, g, h, i)",
		},
		{
			name: "10 parameters",
			ctx: &funcContext{
				funcName:        "TenParams",
				paramNames:      []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"},
				errorResultName: "err",
			},
			wantResult: "defer errs.WrapWith10FuncParams(&err, a, b, c, d, e, f, g, h, i, j)",
		},
		{
			name: "11 parameters (variadic fallback)",
			ctx: &funcContext{
				funcName:        "ElevenParams",
				paramNames:      []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k"},
				errorResultName: "err",
			},
			wantResult: "defer errs.WrapWithFuncParams(&err, a, b, c, d, e, f, g, h, i, j, k)",
		},
		{
			name: "12 parameters (variadic fallback)",
			ctx: &funcContext{
				funcName:        "TwelveParams",
				paramNames:      []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l"},
				errorResultName: "err",
			},
			wantResult: "defer errs.WrapWithFuncParams(&err, a, b, c, d, e, f, g, h, i, j, k, l)",
		},
		{
			name: "custom error result name",
			ctx: &funcContext{
				funcName:        "CustomErr",
				paramNames:      []string{"x", "y"},
				errorResultName: "myError",
			},
			wantResult: "defer errs.WrapWith2FuncParams(&myError, x, y)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateWrapStatement(tt.ctx)
			assert.Equal(t, tt.wantResult, result)
		})
	}
}
