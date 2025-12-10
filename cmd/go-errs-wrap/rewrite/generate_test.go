package rewrite

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateWrapStatement(t *testing.T) {
	tests := []struct {
		name       string
		fun        *funcInfo
		wantResult string
	}{
		{
			name: "0 parameters",
			fun: &funcInfo{
				funcName:        "NoParams",
				paramNames:      nil,
				errorResultName: "err",
			},
			wantResult: "defer errs.WrapWith0FuncParams(&err)",
		},
		{
			name: "1 parameter",
			fun: &funcInfo{
				funcName:        "OneParam",
				paramNames:      []string{"ctx"},
				errorResultName: "err",
			},
			wantResult: "defer errs.WrapWith1FuncParam(&err, ctx)",
		},
		{
			name: "2 parameters",
			fun: &funcInfo{
				funcName:        "TwoParams",
				paramNames:      []string{"ctx", "id"},
				errorResultName: "err",
			},
			wantResult: "defer errs.WrapWith2FuncParams(&err, ctx, id)",
		},
		{
			name: "3 parameters",
			fun: &funcInfo{
				funcName:        "ThreeParams",
				paramNames:      []string{"ctx", "id", "name"},
				errorResultName: "err",
			},
			wantResult: "defer errs.WrapWith3FuncParams(&err, ctx, id, name)",
		},
		{
			name: "4 parameters",
			fun: &funcInfo{
				funcName:        "FourParams",
				paramNames:      []string{"a", "b", "c", "d"},
				errorResultName: "err",
			},
			wantResult: "defer errs.WrapWith4FuncParams(&err, a, b, c, d)",
		},
		{
			name: "5 parameters",
			fun: &funcInfo{
				funcName:        "FiveParams",
				paramNames:      []string{"a", "b", "c", "d", "e"},
				errorResultName: "err",
			},
			wantResult: "defer errs.WrapWith5FuncParams(&err, a, b, c, d, e)",
		},
		{
			name: "6 parameters",
			fun: &funcInfo{
				funcName:        "SixParams",
				paramNames:      []string{"a", "b", "c", "d", "e", "f"},
				errorResultName: "err",
			},
			wantResult: "defer errs.WrapWith6FuncParams(&err, a, b, c, d, e, f)",
		},
		{
			name: "7 parameters",
			fun: &funcInfo{
				funcName:        "SevenParams",
				paramNames:      []string{"a", "b", "c", "d", "e", "f", "g"},
				errorResultName: "err",
			},
			wantResult: "defer errs.WrapWith7FuncParams(&err, a, b, c, d, e, f, g)",
		},
		{
			name: "8 parameters",
			fun: &funcInfo{
				funcName:        "EightParams",
				paramNames:      []string{"a", "b", "c", "d", "e", "f", "g", "h"},
				errorResultName: "err",
			},
			wantResult: "defer errs.WrapWith8FuncParams(&err, a, b, c, d, e, f, g, h)",
		},
		{
			name: "9 parameters",
			fun: &funcInfo{
				funcName:        "NineParams",
				paramNames:      []string{"a", "b", "c", "d", "e", "f", "g", "h", "i"},
				errorResultName: "err",
			},
			wantResult: "defer errs.WrapWith9FuncParams(&err, a, b, c, d, e, f, g, h, i)",
		},
		{
			name: "10 parameters",
			fun: &funcInfo{
				funcName:        "TenParams",
				paramNames:      []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j"},
				errorResultName: "err",
			},
			wantResult: "defer errs.WrapWith10FuncParams(&err, a, b, c, d, e, f, g, h, i, j)",
		},
		{
			name: "11 parameters (variadic fallback)",
			fun: &funcInfo{
				funcName:        "ElevenParams",
				paramNames:      []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k"},
				errorResultName: "err",
			},
			wantResult: "defer errs.WrapWithFuncParams(&err, a, b, c, d, e, f, g, h, i, j, k)",
		},
		{
			name: "12 parameters (variadic fallback)",
			fun: &funcInfo{
				funcName:        "TwelveParams",
				paramNames:      []string{"a", "b", "c", "d", "e", "f", "g", "h", "i", "j", "k", "l"},
				errorResultName: "err",
			},
			wantResult: "defer errs.WrapWithFuncParams(&err, a, b, c, d, e, f, g, h, i, j, k, l)",
		},
		{
			name: "custom error result name",
			fun: &funcInfo{
				funcName:        "CustomErr",
				paramNames:      []string{"x", "y"},
				errorResultName: "myError",
			},
			wantResult: "defer errs.WrapWith2FuncParams(&myError, x, y)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := generateWrapStatement(tt.fun)
			assert.Equal(t, tt.wantResult, result)
		})
	}
}
