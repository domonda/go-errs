package rewrite

import (
	"go/ast"
	"go/token"
	"path"
	"strconv"
	"strings"
	"unicode"
)

// funcInfo holds information about a function needed to generate
// the defer errs.WrapWithFuncParams statement.
type funcInfo struct {
	funcName        string
	paramNames      []string
	keepSecretNames map[string]bool // parameter names that should be wrapped with errs.KeepSecret
	errorResultName string
	errsName        string // name to reference the go-errs package, "errs" if empty
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
	// If no import found, still check for "errs" as default,
	// unless another imported package has that name
	if len(aliases) == 0 && !importsOtherErrsPackage(file) {
		aliases["errs"] = true
	}
	return aliases
}

// importsOtherErrsPackage reports whether file imports a package
// other than go-errs under the name "errs".
func importsOtherErrsPackage(file *ast.File) bool {
	return otherImportNames(file)["errs"]
}

// otherImportNames returns the names under which file imports
// packages other than go-errs, without blank and dot imports.
// For an unnamed import the name is guessed from the import path.
func otherImportNames(file *ast.File) map[string]bool {
	names := make(map[string]bool)
	for _, imp := range file.Imports {
		if imp.Path.Value == errsImportPath {
			continue
		}
		switch {
		case imp.Name != nil:
			names[imp.Name.Name] = true
		default:
			if importPath, err := strconv.Unquote(imp.Path.Value); err == nil {
				names[assumedPackageName(importPath)] = true
			}
		}
	}
	delete(names, "_")
	delete(names, ".")
	return names
}

// assumedPackageName guesses the package name of an import path
// like goimports does: the last path element without a major version
// element, a "go-" prefix and anything after the first character
// that is not valid in an identifier.
func assumedPackageName(importPath string) string {
	base := path.Base(importPath)
	if strings.HasPrefix(base, "v") {
		if _, err := strconv.Atoi(base[1:]); err == nil {
			base = path.Base(path.Dir(importPath))
		}
	}
	base = strings.TrimPrefix(base, "go-")
	if i := strings.IndexFunc(base, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r) && r != '_'
	}); i >= 0 {
		base = base[:i]
	}
	return base
}

// errsImportName returns the name under which file imports the go-errs package
// for use in generated code, preferring an unnamed import over an aliased one.
// It returns an empty string if file has no such import,
// also if it only has a blank or dot import.
func errsImportName(file *ast.File) (name string) {
	for _, imp := range file.Imports {
		if imp.Path.Value != errsImportPath {
			continue
		}
		if imp.Name == nil {
			return "errs"
		}
		if imp.Name.Name != "_" && imp.Name.Name != "." {
			name = imp.Name.Name
		}
	}
	return name
}

// isShadowedAt reports whether name is declared in file where it is in scope
// at pos, so that it does not refer to an imported package there.
// Checked are the receivers, type parameters, parameters and results
// of the functions enclosing pos, and the variables, constants and types
// declared before pos in the blocks and statements enclosing it.
// Package level declarations are not checked, they conflict with an import.
func isShadowedAt(file *ast.File, pos token.Pos, name string) (shadowed bool) {
	declares := func(idents ...*ast.Ident) {
		for _, ident := range idents {
			if ident != nil && ident.Name == name {
				shadowed = true
			}
		}
	}
	declaresFields := func(fieldLists ...*ast.FieldList) {
		for _, fieldList := range fieldLists {
			if fieldList != nil {
				for _, field := range fieldList.List {
					declares(field.Names...)
				}
			}
		}
	}
	// declaresBefore checks the declarations of a statement
	// whose scope continues after the statement and that ends before pos
	declaresBefore := func(stmt ast.Stmt) {
		if stmt == nil || stmt.End() > pos {
			return
		}
		switch stmt := stmt.(type) {
		case *ast.AssignStmt:
			if stmt.Tok == token.DEFINE {
				for _, expr := range stmt.Lhs {
					if ident, ok := expr.(*ast.Ident); ok {
						declares(ident)
					}
				}
			}
		case *ast.DeclStmt:
			if genDecl, ok := stmt.Decl.(*ast.GenDecl); ok {
				for _, spec := range genDecl.Specs {
					switch spec := spec.(type) {
					case *ast.ValueSpec:
						declares(spec.Names...)
					case *ast.TypeSpec:
						declares(spec.Name)
					}
				}
			}
		}
	}

	ast.Inspect(file, func(n ast.Node) bool {
		if shadowed || n == nil || pos < n.Pos() || pos > n.End() {
			return false
		}
		switch node := n.(type) {
		case *ast.FuncDecl:
			declaresFields(node.Recv, node.Type.TypeParams, node.Type.Params, node.Type.Results)
		case *ast.FuncLit:
			declaresFields(node.Type.Params, node.Type.Results)
		case *ast.BlockStmt:
			for _, stmt := range node.List {
				declaresBefore(stmt)
			}
		case *ast.CaseClause:
			for _, stmt := range node.Body {
				declaresBefore(stmt)
			}
		case *ast.CommClause:
			declaresBefore(node.Comm)
			for _, stmt := range node.Body {
				declaresBefore(stmt)
			}
		case *ast.IfStmt:
			declaresBefore(node.Init)
		case *ast.ForStmt:
			declaresBefore(node.Init)
		case *ast.SwitchStmt:
			declaresBefore(node.Init)
		case *ast.TypeSwitchStmt:
			declaresBefore(node.Init)
			declaresBefore(node.Assign)
		case *ast.RangeStmt:
			if node.Tok == token.DEFINE && pos >= node.Body.Pos() {
				for _, expr := range []ast.Expr{node.Key, node.Value} {
					if ident, ok := expr.(*ast.Ident); ok {
						declares(ident)
					}
				}
			}
		}
		return !shadowed
	})
	return shadowed
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
		if !ok || !errsAliases[ident.Name] || ident.Obj != nil || sel.Sel.Name != "KeepSecret" {
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
