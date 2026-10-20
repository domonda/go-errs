package rewrite

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"github.com/ungerik/go-astvisit"
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

// errsImportNames returns the names under which file imports the go-errs
// package for use in code, the name of an unnamed import first, without
// blank and dot imports, and whether file imports go-errs at all.
func errsImportNames(file *ast.File) (names []string, imported bool) {
	for _, imp := range file.Imports {
		if imp.Path.Value != errsImportPath {
			continue
		}
		imported = true
		switch {
		case imp.Name == nil:
			// The default import name is errs, the package name of go-errs
			names = slices.Insert(names, 0, "errs")
		case imp.Name.Name != "_" && imp.Name.Name != ".":
			names = append(names, imp.Name.Name)
		}
	}
	return names, imported
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

// packageNames returns the names declared at package level by the files
// of pkg and by the test files in its directory that belong to the package,
// because no file of the package can import a package under such a name.
func packageNames(pkg *astvisit.ParsedPackage) (map[string]bool, error) {
	names := make(map[string]bool)
	for _, file := range pkg.Files {
		addPackageNames(file, names)
	}
	// Test files are not parsed for the rewrite, but their declarations
	// conflict with an import too when the package is tested
	entries, err := os.ReadDir(pkg.Dir)
	if err != nil {
		return nil, err
	}
	fset := token.NewFileSet()
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), "_test.go") {
			continue
		}
		// A test file that can't be parsed doesn't compile anyway,
		// the declarations parsed before the syntax error are still used
		file, _ := parser.ParseFile(fset, filepath.Join(pkg.Dir, entry.Name()), nil, parser.SkipObjectResolution)
		if file != nil && file.Name != nil && file.Name.Name == pkg.Name {
			addPackageNames(file, names)
		}
	}
	return names, nil
}

// addPackageNames adds the names declared at package level by file to names.
func addPackageNames(file *ast.File, names map[string]bool) {
	for _, decl := range file.Decls {
		switch decl := decl.(type) {
		case *ast.FuncDecl:
			// Methods are not declared in the package block
			if decl.Recv == nil {
				names[decl.Name.Name] = true
			}
		case *ast.GenDecl:
			for _, spec := range decl.Specs {
				switch spec := spec.(type) {
				case *ast.ValueSpec:
					for _, name := range spec.Names {
						names[name.Name] = true
					}
				case *ast.TypeSpec:
					names[spec.Name.Name] = true
				}
			}
		}
	}
}

// firstUnshadowedName returns the first of names that is not shadowed
// in file at pos, see isShadowedAt, or an empty string if all of them are.
func firstUnshadowedName(file *ast.File, pos token.Pos, names []string) string {
	for _, name := range names {
		if !isShadowedAt(file, pos, name) {
			return name
		}
	}
	return ""
}

// newErrsImportName returns the name for a new import of go-errs into file
// that is free at all positions: the first of errs, goerrs, goerrs2 and so on
// that isReserved doesn't report as name of another import or a package level
// declaration, and that is not shadowed at any of positions, see isShadowedAt.
func newErrsImportName(file *ast.File, positions []token.Pos, isReserved func(name string) bool) string {
	isFree := func(name string) bool {
		if isReserved(name) {
			return false
		}
		for _, pos := range positions {
			if isShadowedAt(file, pos, name) {
				return false
			}
		}
		return true
	}
	if isFree("errs") {
		return "errs"
	}
	name := "goerrs"
	for i := 2; !isFree(name); i++ {
		name = "goerrs" + strconv.Itoa(i)
	}
	return name
}

// isShadowedAt reports whether name is declared in file where it is in scope
// at pos, so that it does not refer to an imported package there.
// Checked are the receivers, type parameters, parameters and results
// of the functions enclosing pos, and the variables, constants and types
// declared before pos in the blocks and statements enclosing it.
// Package level declarations are not checked, they conflict with an import
// in every file of the package, see packageNames.
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

		if _, name := errsSelector(call.Fun, errsAliases); name != "KeepSecret" {
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
