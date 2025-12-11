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
	keepSecretNames map[string]bool // parameter names that should be wrapped with errs.KeepSecret
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

// errsImportPath is the import path for the go-errs package.
const errsImportPath = `"github.com/domonda/go-errs"`

// extractErrsAliases returns a set of all names (aliases or default "errs")
// used to import the go-errs package in the given file.
func extractErrsAliases(file *ast.File) map[string]bool {
	aliases := make(map[string]bool)
	for _, imp := range file.Imports {
		if imp.Path.Value != errsImportPath {
			continue
		}
		if imp.Name != nil {
			// Explicit alias: import alias "github.com/domonda/go-errs"
			aliases[imp.Name.Name] = true
		} else {
			// Default import name is "errs" (last element of path)
			aliases["errs"] = true
		}
	}
	// If no import found, still check for "errs" as default
	if len(aliases) == 0 {
		aliases["errs"] = true
	}
	return aliases
}

// extractKeepSecretParams extracts the parameter names that are wrapped with
// errs.KeepSecret() from an existing defer errs.Wrap* statement.
// This allows the replacement to preserve the KeepSecret wrapping that developers
// have intentionally added to prevent secrets from appearing in callstacks and logs.
//
// The errsAliases parameter contains all possible names used to import go-errs,
// to handle cases where the import is aliased (e.g., import e "github.com/domonda/go-errs").
func extractKeepSecretParams(stmt *ast.DeferStmt, errsAliases map[string]bool) map[string]bool {
	result := make(map[string]bool)

	_, ok := stmt.Call.Fun.(*ast.SelectorExpr)
	if !ok {
		return result
	}

	// Skip the first argument (which is &err)
	for i, arg := range stmt.Call.Args {
		if i == 0 {
			continue // Skip &err
		}

		// Check if the argument is errs.KeepSecret(paramName) or alias.KeepSecret(paramName)
		call, ok := arg.(*ast.CallExpr)
		if !ok {
			continue
		}

		sel, ok := call.Fun.(*ast.SelectorExpr)
		if !ok {
			continue
		}

		ident, ok := sel.X.(*ast.Ident)
		if !ok || !errsAliases[ident.Name] || sel.Sel.Name != "KeepSecret" {
			continue
		}

		// Found KeepSecret(...), extract the inner parameter name
		if len(call.Args) != 1 {
			continue
		}

		paramIdent, ok := call.Args[0].(*ast.Ident)
		if !ok {
			continue
		}

		result[paramIdent.Name] = true
	}

	return result
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
