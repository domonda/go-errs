package rewrite

import (
	"fmt"
	"strings"
)

// formatParamWithKeepSecret formats a parameter name, wrapping it with
// errs.KeepSecret() if it's in the keepSecretNames set.
func formatParamWithKeepSecret(paramName string, keepSecretNames map[string]bool) string {
	if keepSecretNames != nil && keepSecretNames[paramName] {
		return fmt.Sprintf("errs.KeepSecret(%s)", paramName)
	}
	return paramName
}

// formatParams formats all parameter names, applying KeepSecret wrapping where needed.
func formatParams(paramNames []string, keepSecretNames map[string]bool) string {
	formatted := make([]string, len(paramNames))
	for i, name := range paramNames {
		formatted[i] = formatParamWithKeepSecret(name, keepSecretNames)
	}
	return strings.Join(formatted, ", ")
}

// generateWrapStatement generates the appropriate defer errs.WrapWith*FuncParams
// statement based on the function context.
//
// It uses the optimized WrapWith0FuncParams through WrapWith10FuncParams for
// 0-10 parameters, and falls back to the variadic WrapWithFuncParams for 11+.
//
// If keepSecretNames contains any parameter names, those parameters will be
// wrapped with errs.KeepSecret() to prevent secrets from appearing in logs.
func generateWrapStatement(fun *funcInfo) string {
	switch numParams := len(fun.paramNames); {
	case numParams == 0:
		return fmt.Sprintf("defer errs.WrapWith0FuncParams(&%s)",
			fun.errorResultName,
		)
	case numParams == 1:
		return fmt.Sprintf("defer errs.WrapWith1FuncParam(&%s, %s)",
			fun.errorResultName,
			formatParamWithKeepSecret(fun.paramNames[0], fun.keepSecretNames),
		)
	case numParams > 10:
		// 11+ parameters: use variadic version
		return generateVariadicWrapStatement(fun)
	default:
		return fmt.Sprintf("defer errs.WrapWith%dFuncParams(&%s, %s)",
			numParams,
			fun.errorResultName,
			formatParams(fun.paramNames, fun.keepSecretNames),
		)
	}
}

func generateVariadicWrapStatement(fun *funcInfo) string {
	return fmt.Sprintf("defer errs.WrapWithFuncParams(&%s, %s)",
		fun.errorResultName,
		formatParams(fun.paramNames, fun.keepSecretNames),
	)
}
