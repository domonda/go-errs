package rewrite

import (
	"fmt"
	"strings"
)

// errsPkg returns the name to use for the go-errs package in generated code.
func (fun *funcInfo) errsPkg() string {
	if fun.errsName == "" {
		return "errs"
	}
	return fun.errsName
}

// formatParamWithKeepSecret formats a parameter name, wrapping it with
// errs.KeepSecret() if it's in the keepSecretNames set of fun.
func (fun *funcInfo) formatParamWithKeepSecret(paramName string) string {
	if fun.keepSecretNames != nil && fun.keepSecretNames[paramName] {
		return fmt.Sprintf("%s.KeepSecret(%s)", fun.errsPkg(), paramName)
	}
	return paramName
}

// formatParams formats all parameter names of fun, applying KeepSecret wrapping where needed.
func (fun *funcInfo) formatParams() string {
	formatted := make([]string, len(fun.paramNames))
	for i, name := range fun.paramNames {
		formatted[i] = fun.formatParamWithKeepSecret(name)
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
// The go-errs package is referenced by fun.errsName.
func generateWrapStatement(fun *funcInfo) string {
	switch numParams := len(fun.paramNames); {
	case numParams == 0:
		return fmt.Sprintf("defer %s.WrapWith0FuncParams(&%s)",
			fun.errsPkg(),
			fun.errorResultName,
		)
	case numParams == 1:
		return fmt.Sprintf("defer %s.WrapWith1FuncParam(&%s, %s)",
			fun.errsPkg(),
			fun.errorResultName,
			fun.formatParamWithKeepSecret(fun.paramNames[0]),
		)
	case numParams > 10:
		// 11+ parameters: use variadic version
		return generateVariadicWrapStatement(fun)
	default:
		return fmt.Sprintf("defer %s.WrapWith%dFuncParams(&%s, %s)",
			fun.errsPkg(),
			numParams,
			fun.errorResultName,
			fun.formatParams(),
		)
	}
}

func generateVariadicWrapStatement(fun *funcInfo) string {
	if len(fun.paramNames) == 0 {
		return fmt.Sprintf("defer %s.WrapWithFuncParams(&%s)",
			fun.errsPkg(),
			fun.errorResultName,
		)
	}
	return fmt.Sprintf("defer %s.WrapWithFuncParams(&%s, %s)",
		fun.errsPkg(),
		fun.errorResultName,
		fun.formatParams(),
	)
}
