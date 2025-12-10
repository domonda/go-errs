package rewrite

import (
	"go/ast"
	"go/token"
)

// funcInfo holds information about a function needed to generate
// the defer errs.WrapWithFuncParams statement.
type funcInfo struct {
	funcName        string
	paramNames      []string
	errorResultName string
	startPos        token.Pos
	endPos          token.Pos
}

// extractFuncInfo extracts the function information from a function type.
func extractFuncInfo(funcType *ast.FuncType, funcName string, startPos, endPos token.Pos) *funcInfo {
	fun := &funcInfo{
		funcName: funcName,
		startPos: startPos,
		endPos:   endPos,
	}

	// Extract parameter names (excluding blank identifiers)
	if funcType.Params != nil {
		for _, field := range funcType.Params.List {
			for _, name := range field.Names {
				if name.Name != "_" {
					fun.paramNames = append(fun.paramNames, name.Name)
				}
			}
		}
	}

	// Find the named error result (use the last one if multiple)
	if funcType.Results != nil {
		for _, field := range funcType.Results.List {
			if isErrorType(field.Type) {
				for _, name := range field.Names {
					if name.Name != "_" {
						fun.errorResultName = name.Name
					}
				}
			}
		}
	}

	return fun
}

// isErrorType checks if the type expression is the error type.
func isErrorType(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	if !ok {
		return false
	}
	return ident.Name == "error"
}

// findEnclosingFuncForPos walks the AST to find the innermost function containing the given position.
// This is used for finding the function context for comments, which aren't visited
// during the normal AST traversal.
func findEnclosingFuncForPos(file *ast.File, pos token.Pos) (result *funcInfo) {
	ast.Inspect(file, func(n ast.Node) bool {
		if n == nil || pos < n.Pos() || pos > n.End() {
			return false
		}

		switch node := n.(type) {
		case *ast.FuncDecl:
			result = extractFuncInfo(node.Type, node.Name.Name, node.Pos(), node.End())
		case *ast.FuncLit:
			result = extractFuncInfo(node.Type, "(anonymous)", node.Pos(), node.End())
		}
		// Continue traversing to find nested function literals.
		// If a nested function also contains the position,
		// result will be reassigned to the innermost one.
		return true
	})

	return result
}
