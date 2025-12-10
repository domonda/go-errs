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
func generateWrapStatement(ctx *funcContext) string {
	numParams := len(ctx.paramNames)

	switch numParams {
	case 0:
		return fmt.Sprintf("defer errs.WrapWith0FuncParams(&%s)", ctx.errorResultName)
	case 1:
		return fmt.Sprintf("defer errs.WrapWith1FuncParam(&%s, %s)",
			ctx.errorResultName, ctx.paramNames[0])
	case 2:
		return fmt.Sprintf("defer errs.WrapWith2FuncParams(&%s, %s)",
			ctx.errorResultName, strings.Join(ctx.paramNames, ", "))
	case 3:
		return fmt.Sprintf("defer errs.WrapWith3FuncParams(&%s, %s)",
			ctx.errorResultName, strings.Join(ctx.paramNames, ", "))
	case 4:
		return fmt.Sprintf("defer errs.WrapWith4FuncParams(&%s, %s)",
			ctx.errorResultName, strings.Join(ctx.paramNames, ", "))
	case 5:
		return fmt.Sprintf("defer errs.WrapWith5FuncParams(&%s, %s)",
			ctx.errorResultName, strings.Join(ctx.paramNames, ", "))
	case 6:
		return fmt.Sprintf("defer errs.WrapWith6FuncParams(&%s, %s)",
			ctx.errorResultName, strings.Join(ctx.paramNames, ", "))
	case 7:
		return fmt.Sprintf("defer errs.WrapWith7FuncParams(&%s, %s)",
			ctx.errorResultName, strings.Join(ctx.paramNames, ", "))
	case 8:
		return fmt.Sprintf("defer errs.WrapWith8FuncParams(&%s, %s)",
			ctx.errorResultName, strings.Join(ctx.paramNames, ", "))
	case 9:
		return fmt.Sprintf("defer errs.WrapWith9FuncParams(&%s, %s)",
			ctx.errorResultName, strings.Join(ctx.paramNames, ", "))
	case 10:
		return fmt.Sprintf("defer errs.WrapWith10FuncParams(&%s, %s)",
			ctx.errorResultName, strings.Join(ctx.paramNames, ", "))
	default:
		// 11+ parameters: use variadic version
		return fmt.Sprintf("defer errs.WrapWithFuncParams(&%s, %s)",
			ctx.errorResultName, strings.Join(ctx.paramNames, ", "))
	}
}
