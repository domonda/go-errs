package rewrite

import (
	"fmt"
	"strings"
)

// generateWrapStatement generates the appropriate defer errs.WrapWith*FuncParams
// statement based on the function context.
//
// It uses the optimized WrapWith0FuncParams through WrapWith10FuncParams for
// 0-10 parameters, and falls back to the variadic WrapWithFuncParams for 11+.
func generateWrapStatement(fun *funcInfo) string {
	switch numParams := len(fun.paramNames); {
	case numParams == 0:
		return fmt.Sprintf("defer errs.WrapWith0FuncParams(&%s)",
			fun.errorResultName,
		)
	case numParams == 1:
		return fmt.Sprintf("defer errs.WrapWith1FuncParam(&%s, %s)",
			fun.errorResultName, fun.paramNames[0],
		)
	case numParams > 10:
		// 11+ parameters: use variadic version
		return generateVariadicWrapStatement(fun)
	default:
		return fmt.Sprintf("defer errs.WrapWith%dFuncParams(&%s, %s)",
			numParams, fun.errorResultName, strings.Join(fun.paramNames, ", "),
		)
	}
}

func generateVariadicWrapStatement(fun *funcInfo) string {
	return fmt.Sprintf("defer errs.WrapWithFuncParams(&%s, %s)",
		fun.errorResultName, strings.Join(fun.paramNames, ", "),
	)
}
