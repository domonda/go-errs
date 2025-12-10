package rewrite

import (
	"go/ast"
	"go/token"
)

// funcContext holds information about a function needed to generate
// the defer errs.WrapWithFuncParams statement.
type funcContext struct {
	funcName        string
	paramNames      []string
	errorResultName string
	startPos        token.Pos
	endPos          token.Pos
}

// extractFuncContext extracts the function context from a function type.
func extractFuncContext(funcType *ast.FuncType, funcName string, startPos, endPos token.Pos) *funcContext {
	ctx := &funcContext{
		funcName: funcName,
		startPos: startPos,
		endPos:   endPos,
	}

	// Extract parameter names (excluding blank identifiers)
	if funcType.Params != nil {
		for _, field := range funcType.Params.List {
			for _, name := range field.Names {
				if name.Name != "_" {
					ctx.paramNames = append(ctx.paramNames, name.Name)
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
						ctx.errorResultName = name.Name
					}
				}
			}
		}
	}

	return ctx
}

// isErrorType checks if the type expression is the error type.
func isErrorType(expr ast.Expr) bool {
	ident, ok := expr.(*ast.Ident)
	if !ok {
		return false
	}
	return ident.Name == "error"
}

// findEnclosingFunc finds the innermost function context that contains the given position.
func findEnclosingFunc(funcStack []*funcContext, pos token.Pos) *funcContext {
	// Return the innermost (last) function that contains this position
	for i := len(funcStack) - 1; i >= 0; i-- {
		ctx := funcStack[i]
		if pos >= ctx.startPos && pos <= ctx.endPos {
			return ctx
		}
	}
	return nil
}

// findEnclosingFuncForPos walks the AST to find the function containing the given position.
// This is used for finding the function context for comments, which aren't visited
// during the normal AST traversal.
func findEnclosingFuncForPos(file *ast.File, pos token.Pos) *funcContext {
	var result *funcContext

	ast.Inspect(file, func(n ast.Node) bool {
		if n == nil {
			return true
		}

		switch node := n.(type) {
		case *ast.FuncDecl:
			if pos >= node.Pos() && pos <= node.End() {
				result = extractFuncContext(node.Type, node.Name.Name, node.Pos(), node.End())
				// Check for nested function literals
				return true
			}
		case *ast.FuncLit:
			if pos >= node.Pos() && pos <= node.End() {
				result = extractFuncContext(node.Type, "(anonymous)", node.Pos(), node.End())
				// Check for nested function literals
				return true
			}
		}

		return true
	})

	return result
}
