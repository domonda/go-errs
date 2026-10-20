// Package rewrite provides functionality to remove, replace, or insert
// defer errs.Wrap statements and //#wrap-result-err marker comments
// in Go source files.
package rewrite

import (
	"bytes"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ungerik/go-astvisit"
	"golang.org/x/tools/go/ast/astutil"

	"github.com/domonda/go-errs"
)

// processMode specifies the operation mode for processing files.
type processMode int

const (
	modeRemove  processMode = iota // Remove all defer errs.Wrap statements
	modeReplace                    // Replace existing defer errs.Wrap statements
	modeInsert                     // Insert new defer errs.Wrap statements where missing
)

// Remove removes all defer errs.Wrap statements and //#wrap-result-err
// marker comments from Go source files at the given path.
//
// If outPath is empty, files are modified in place.
// If outPath is specified, results are written there instead.
// If recursive is true, subdirectories are processed recursively.
// If validate is true, no files are modified; instead, any defer errs.Wrap
// statements or markers found are reported to stderr and the function returns
// an error if any are found. When validate is true, outPath is ignored.
func Remove(sourcePath, outPath string, recursive, validate bool, verboseOut io.Writer) (err error) {
	defer errs.WrapWithFuncParams(&err, sourcePath, outPath, recursive, validate, verboseOut)

	return process(sourcePath, outPath, recursive, false, validate, verboseOut, modeRemove)
}

// Replace replaces all defer errs.Wrap statements and //#wrap-result-err
// marker comments with properly generated defer errs.WrapWithFuncParams
// statements in Go source files at the given path.
//
// If outPath is empty, files are modified in place.
// If outPath is specified, results are written there instead.
// If recursive is true, subdirectories are processed recursively.
// If minVariadic is true, always use specialized WrapWithNFuncParams functions
// instead of preserving existing variadic WrapWithFuncParams calls.
// If validate is true, no files are modified; instead, missing replacements
// are reported to stderr and the function returns an error if any are found.
// When validate is true, outPath is ignored.
func Replace(sourcePath, outPath string, recursive, minVariadic, validate bool, verboseOut io.Writer) (err error) {
	defer errs.WrapWithFuncParams(&err, sourcePath, outPath, recursive, minVariadic, validate, verboseOut)

	return process(sourcePath, outPath, recursive, minVariadic, validate, verboseOut, modeReplace)
}

// Insert inserts defer errs.WrapWith*FuncParams statements at the first line
// of every function that has a named error result but doesn't already have
// a defer errs.Wrap statement. An empty line is added after the inserted statement.
//
// If outPath is empty, files are modified in place.
// If outPath is specified, results are written there instead.
// If recursive is true, subdirectories are processed recursively.
// If minVariadic is true, always use specialized WrapWithNFuncParams functions.
// If validate is true, no files are modified; instead, missing insertions
// are reported to stderr and the function returns an error if any are found.
// When validate is true, outPath is ignored.
func Insert(sourcePath, outPath string, recursive, minVariadic, validate bool, verboseOut io.Writer) (err error) {
	defer errs.WrapWithFuncParams(&err, sourcePath, outPath, recursive, minVariadic, validate, verboseOut)

	return process(sourcePath, outPath, recursive, minVariadic, validate, verboseOut, modeInsert)
}

func process(sourcePath, outPath string, recursive, minVariadic, validate bool, verboseOut io.Writer, mode processMode) (err error) {
	defer errs.WrapWithFuncParams(&err, sourcePath, outPath, recursive, minVariadic, validate, verboseOut, mode)

	sourcePath, err = filepath.Abs(sourcePath)
	if err != nil {
		return err
	}

	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		return err
	}

	// Use "..." suffix for directories to recursively find all packages
	path := sourcePath
	if sourceInfo.IsDir() && recursive {
		path = strings.TrimSuffix(sourcePath, "/") + "/..."
	}

	if validate {
		// Collect the missing changes without modifying files, outPath is ignored
		var validationErrors []string
		_, err = astvisit.RewriteDryRun(path, verboseOut, packageRewriter(minVariadic, mode, verboseOut, &validationErrors))
		if err != nil {
			return err
		}
		if len(validationErrors) > 0 {
			for _, errMsg := range validationErrors {
				fmt.Fprintln(os.Stderr, errMsg)
			}
			return errs.Errorf("found %d missing error wrapper(s)", len(validationErrors))
		}
		return nil
	}

	if outPath == "" {
		// astvisit writes only the changed files of a package
		// after all of them were rewritten, keeping their line endings
		return astvisit.RewritePackages(path, verboseOut, nil, packageRewriter(minVariadic, mode, verboseOut, nil))
	}

	outPath, err = filepath.Abs(outPath)
	if err != nil {
		return err
	}
	if sourceInfo.IsDir() {
		// Writing the output into the source directory, or the source
		// directory into one of its parents, would overwrite source files
		overlap, err := directoriesOverlap(sourcePath, outPath)
		if err != nil {
			return err
		}
		if overlap {
			return errs.Errorf("output path %s must not be the source directory %s, be within it or contain it", outPath, sourcePath)
		}
	}

	// Rewrite all files before writing any output,
	// so an error doesn't leave a partially written output
	rewritten := make(map[string][]byte)
	rewritePackage := packageRewriter(minVariadic, mode, verboseOut, nil)
	_, err = astvisit.RewriteDryRun(path, nil, func(pkg *astvisit.ParsedPackage, verboseOut io.Writer) (map[string][]byte, error) {
		sources, err := rewritePackage(pkg, verboseOut)
		if err != nil {
			return nil, err
		}
		for filePath, source := range sources {
			rewritten[filePath] = withLineEndingsOf(source, pkg.Sources[filePath])
		}
		return sources, nil
	})
	if err != nil {
		return err
	}

	if sourceInfo.IsDir() {
		return writeDirectoryOutput(sourcePath, outPath, recursive, rewritten, verboseOut)
	}
	return writeFileOutput(sourcePath, outPath, rewritten[sourcePath], verboseOut)
}

// packageRewriter returns the astvisit.RewritePackageFunc that rewrites
// the files of a package. If validationErrors is not nil, no file is rewritten,
// instead the missing changes are appended to validationErrors.
func packageRewriter(minVariadic bool, mode processMode, verboseOut io.Writer, validationErrors *[]string) astvisit.RewritePackageFunc {
	return func(pkg *astvisit.ParsedPackage, _ io.Writer) (map[string][]byte, error) {
		rewritten := make(map[string][]byte)
		for _, filePath := range pkg.FilePaths {
			source := pkg.Sources[filePath]
			replacements, imports, err := processFile(pkg.Fset, pkg.Files[filePath], minVariadic, verboseOut, mode)
			if err != nil {
				return nil, err
			}

			// Filter out no-op replacements: a replacement that produces the
			// exact source text already at its node's position would only
			// trigger import reordering and other formatting churn without
			// actually fixing anything. Skipping them here ensures both:
			//   - validate mode reports only genuine issues, not false
			//     positives caused by unrelated formatting differences
			//   - normal mode does not rewrite a file (and reorder its
			//     imports) when every wrap statement is already correct
			filtered := replacements[:0]
			for _, repl := range replacements {
				if repl.Node != nil && !replacementChangesSource(pkg.Fset, source, repl) {
					continue
				}
				filtered = append(filtered, repl)
			}
			if len(filtered) == 0 {
				continue
			}

			if validationErrors != nil {
				for _, repl := range filtered {
					var pos token.Position
					if repl.Node != nil {
						pos = pkg.Fset.Position(repl.Node.Pos())
					}
					desc := repl.DebugID
					if desc == "" {
						desc = "error wrapper"
					}
					*validationErrors = append(*validationErrors, fmt.Sprintf("%s: missing %s", pos, desc))
				}
				continue
			}

			rewritten[filePath], err = rewriteSource(pkg.Fset, filePath, source, filtered, imports, mode)
			if err != nil {
				return nil, err
			}
		}
		return rewritten, nil
	}
}

// rewriteSource returns source of filePath with replacements applied
// and formatted like gofmt. For remove, a go-errs import that is not used
// anymore is removed, for replace and insert the missing imports are added.
func rewriteSource(fset *token.FileSet, filePath string, source []byte, replacements astvisit.NodeReplacements, imports astvisit.Imports, mode processMode) ([]byte, error) {
	rewritten, err := replacements.Apply(fset, source)
	if err != nil {
		return nil, err
	}
	if mode == modeRemove {
		return removeUnusedErrsImports(filePath, rewritten)
	}
	return astvisit.FormatSource(rewritten, imports)
}

// removeUnusedErrsImports removes the imports of go-errs from source
// whose package name is not used anymore and formats it like gofmt.
// Other imports are kept unchanged.
func removeUnusedErrsImports(filePath string, source []byte) ([]byte, error) {
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filePath, source, parser.ParseComments)
	if err != nil {
		return nil, err
	}

	// A selector on an identifier that the parser could not resolve
	// to a declaration of the file uses an imported package
	usedPackages := make(map[string]bool)
	ast.Inspect(file, func(n ast.Node) bool {
		if sel, ok := n.(*ast.SelectorExpr); ok {
			if ident, ok := sel.X.(*ast.Ident); ok && ident.Obj == nil {
				usedPackages[ident.Name] = true
			}
		}
		return true
	})

	var unusedNames []string
	for _, imp := range file.Imports {
		if imp.Path.Value != errsImportPath {
			continue
		}
		switch {
		case imp.Name == nil:
			if !usedPackages["errs"] {
				unusedNames = append(unusedNames, "")
			}
		case imp.Name.Name != "_" && imp.Name.Name != "." && !usedPackages[imp.Name.Name]:
			unusedNames = append(unusedNames, imp.Name.Name)
		}
	}
	for _, name := range unusedNames {
		astutil.DeleteNamedImport(fset, file, name, strings.Trim(errsImportPath, `"`))
	}

	var buf bytes.Buffer
	err = format.Node(&buf, fset, file)
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// withLineEndingsOf returns source with the line endings (LF or CRLF)
// of the first line of original, like astvisit writes rewritten files.
func withLineEndingsOf(source, original []byte) []byte {
	source = bytes.ReplaceAll(source, []byte("\r\n"), []byte("\n"))
	if i := bytes.IndexByte(original, '\n'); i > 0 && original[i-1] == '\r' {
		source = bytes.ReplaceAll(source, []byte("\n"), []byte("\r\n"))
	}
	return source
}

// writeFileOutput writes the rewritten source of sourceFile,
// or its unchanged source if rewritten is nil, to outPath.
// If outPath is an existing directory or doesn't end with .go,
// the file is written with the name of sourceFile into that directory,
// which is created if missing.
func writeFileOutput(sourceFile, outPath string, rewritten []byte, verboseOut io.Writer) error {
	outInfo, err := os.Stat(outPath)
	if err == nil && outInfo.IsDir() {
		outPath = filepath.Join(outPath, filepath.Base(sourceFile))
	} else if !strings.HasSuffix(outPath, ".go") {
		err = os.MkdirAll(outPath, 0750)
		if err != nil {
			return err
		}
		outPath = filepath.Join(outPath, filepath.Base(sourceFile))
	}

	if rewritten == nil {
		// #nosec G304 -- sourceFile is the file path passed by the user
		rewritten, err = os.ReadFile(sourceFile)
		if err != nil {
			return err
		}
	}
	err = writeFile(outPath, bytes.NewReader(rewritten), 0640)
	if err != nil {
		return err
	}
	if verboseOut != nil {
		fmt.Fprintf(verboseOut, "wrote: %s\n", outPath)
	}
	return nil
}

// writeDirectoryOutput copies the files below sourceDir, or without recursive
// only the files directly in sourceDir, to outDir and writes the rewritten
// sources instead of the original ones of the files in rewritten.
// Version control metadata and files that are not regular files
// or symbolic links to regular files are not copied.
// Copied files and directories are writable by the owner,
// so that the output can be written again.
func writeDirectoryOutput(sourceDir, outDir string, recursive bool, rewritten map[string][]byte, verboseOut io.Writer) error {
	// filepath.Walk doesn't follow a symbolic link as root
	root, err := filepath.EvalSymlinks(sourceDir)
	if err != nil {
		return err
	}
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		// astvisit names the files with the unresolved source directory
		sourcePath := filepath.Join(sourceDir, relPath)
		destPath := filepath.Join(outDir, relPath)

		if info.IsDir() {
			if path != root && (!recursive || isVCSMetadata(info.Name())) {
				return filepath.SkipDir
			}
			return os.MkdirAll(destPath, info.Mode().Perm()|0700)
		}
		if isVCSMetadata(info.Name()) {
			// Like the .git file of a git worktree
			return nil
		}

		if info.Mode()&os.ModeSymlink != 0 {
			// Copy the file a symbolic link points to
			info, err = os.Stat(path)
			if err != nil || !info.Mode().IsRegular() {
				if verboseOut != nil {
					fmt.Fprintf(verboseOut, "skipping symbolic link: %s\n", sourcePath)
				}
				return nil
			}
		} else if !info.Mode().IsRegular() {
			if verboseOut != nil {
				fmt.Fprintf(verboseOut, "skipping irregular file: %s\n", sourcePath)
			}
			return nil
		}

		if source, ok := rewritten[sourcePath]; ok {
			if verboseOut != nil {
				fmt.Fprintf(verboseOut, "wrote: %s\n", destPath)
			}
			return writeFile(destPath, bytes.NewReader(source), info.Mode().Perm()|0200)
		}
		if verboseOut != nil {
			fmt.Fprintf(verboseOut, "copying: %s -> %s\n", sourcePath, destPath)
		}
		return copyFile(path, destPath, info.Mode().Perm()|0200)
	})
}

// isVCSMetadata reports whether a file or directory name
// is the metadata of a version control system.
func isVCSMetadata(name string) bool {
	switch name {
	case ".git", ".hg", ".svn":
		return true
	}
	return false
}

// directoriesOverlap reports whether outPath is the directory sourceDir,
// is within it, or contains it. Directories are compared by their identity,
// so that other spellings of the same directory are found too, like another
// letter case on a case-insensitive file system or a path through a link.
func directoriesOverlap(sourceDir, outPath string) (bool, error) {
	sourceInfo, err := os.Stat(sourceDir)
	if err != nil {
		return false, err
	}
	// outPath, or its longest existing prefix, is sourceDir or within it
	for dir := resolvePath(outPath); ; dir = filepath.Dir(dir) {
		if info, err := os.Stat(dir); err == nil && os.SameFile(info, sourceInfo) {
			return true, nil
		}
		if filepath.Dir(dir) == dir {
			break
		}
	}
	// An existing outPath contains sourceDir
	if outInfo, err := os.Stat(outPath); err == nil {
		for dir := filepath.Dir(resolvePath(sourceDir)); ; dir = filepath.Dir(dir) {
			if info, err := os.Stat(dir); err == nil && os.SameFile(info, outInfo) {
				return true, nil
			}
			if filepath.Dir(dir) == dir {
				break
			}
		}
	}
	return false, nil
}

// resolvePath returns the absolute path with all symbolic links of its
// longest existing prefix evaluated, so that its existing parent directories
// are the actual ones, even if path does not exist yet.
func resolvePath(path string) string {
	resolved, err := filepath.EvalSymlinks(path)
	if err == nil {
		return resolved
	}
	parent := filepath.Dir(path)
	if parent == path {
		return path
	}
	return filepath.Join(resolvePath(parent), filepath.Base(path))
}

func processFile(fset *token.FileSet, astFile *ast.File, minVariadic bool, verboseOut io.Writer, mode processMode) (replacements astvisit.NodeReplacements, imports astvisit.Imports, err error) {
	defer errs.WrapWithFuncParams(&err, fset, astFile, minVariadic, verboseOut, mode)

	imports = make(astvisit.Imports)

	// Extract all aliases used to import go-errs (for detecting KeepSecret calls)
	errsAliases := extractErrsAliases(astFile)
	// Generated statements use the name of an existing go-errs import.
	// If the file has no usable one, or a declaration shadows its name
	// where a statement is generated, go-errs is imported under a name
	// that is free there: unnamed if no other package is imported as errs,
	// else as goerrs, goerrs2 and so on.
	errsName := errsImportName(astFile)
	otherErrsPackage := importsOtherErrsPackage(astFile)
	addedErrsNames := make(map[string]bool)
	errsNameAt := func(pos token.Pos) string {
		if errsName != "" && !isShadowedAt(astFile, pos, errsName) {
			return errsName
		}
		name := "errs"
		if otherErrsPackage || isShadowedAt(astFile, pos, name) {
			name = "goerrs"
			for i := 2; isShadowedAt(astFile, pos, name); i++ {
				name = "goerrs" + strconv.Itoa(i)
			}
		}
		addedErrsNames[name] = true
		return name
	}

	// For insert mode, track which functions already have defer errs.Wrap
	funcsWithWrap := make(map[token.Pos]bool)
	if mode == modeInsert {
		ast.Inspect(astFile, func(n ast.Node) bool {
			if n == nil {
				return true
			}
			deferStmt, ok := n.(*ast.DeferStmt)
			if !ok || !isDeferErrsWrap(deferStmt, errsAliases) {
				return true
			}
			// Find enclosing function and mark it as having a wrap
			fun := findEnclosingFuncForPos(astFile, deferStmt.Pos())
			if fun != nil {
				funcsWithWrap[fun.startPos] = true
			}
			return true
		})
	}

	// Visit all nodes to find defer statements (for remove/replace modes)
	if mode != modeInsert {
		ast.Inspect(astFile, func(n ast.Node) bool {
			if n == nil {
				return true
			}

			deferStmt, ok := n.(*ast.DeferStmt)
			if !ok || !isDeferErrsWrap(deferStmt, errsAliases) {
				return true
			}

			if mode == modeRemove {
				if verboseOut != nil {
					fmt.Fprintf(verboseOut, "%s: removing defer errs.Wrap\n", fset.Position(deferStmt.Pos()))
				}
				replacements.AddRemoval(deferStmt, "remove defer errs.Wrap")
				return true
			}

			// Find enclosing function for replacement
			fun := findEnclosingFuncForPos(astFile, deferStmt.Pos())
			if fun == nil {
				fmt.Fprintf(os.Stderr, "warning: %s: defer statement not inside a function\n",
					fset.Position(deferStmt.Pos()),
				)
				return true
			}
			if fun.errorResultName == "" {
				fmt.Fprintf(os.Stderr, "warning: %s: function %s has no named error result, skipping\n",
					fset.Position(deferStmt.Pos()), fun.funcName,
				)
				return true
			}

			// Extract KeepSecret-wrapped parameters from the existing defer statement
			// so they can be preserved in the replacement
			fun.keepSecretNames = extractKeepSecretParams(deferStmt, errsAliases)
			fun.errsName = errsNameAt(deferStmt.Pos())

			// If already using variadic WrapWithFuncParams and minVariadic is false, keep it variadic
			var replacement string
			if !minVariadic && isVariadicWrapWithFuncParams(deferStmt, errsAliases) {
				replacement = generateVariadicWrapStatement(fun)
			} else {
				replacement = generateWrapStatement(fun)
			}

			if verboseOut != nil {
				fmt.Fprintf(verboseOut, "%s: replacing defer errs.Wrap with %s\n", fset.Position(deferStmt.Pos()), replacement)
			}
			replacements.AddReplacement(deferStmt, replacement, "replace defer errs.Wrap")

			return true
		})
	}

	// Process comments for //#wrap-result-err markers (for remove/replace modes)
	if mode != modeInsert {
		for _, commentGroup := range astFile.Comments {
			for _, comment := range commentGroup.List {
				if !strings.HasPrefix(comment.Text, "//") {
					continue
				}
				trimmed := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
				if trimmed != "#wrap-result-err" {
					continue
				}

				if mode == modeRemove {
					if verboseOut != nil {
						fmt.Fprintf(verboseOut, "%s: removing //#wrap-result-err\n", fset.Position(comment.Pos()))
					}
					replacements.AddRemoval(comment, "remove //#wrap-result-err")
					continue
				}

				// Find enclosing function for this comment
				ctx := findEnclosingFuncForPos(astFile, comment.Pos())
				if ctx == nil {
					fmt.Fprintf(os.Stderr, "warning: %s: //#wrap-result-err not inside a function\n",
						fset.Position(comment.Pos()),
					)
					continue
				}
				if ctx.errorResultName == "" {
					fmt.Fprintf(os.Stderr, "warning: %s: function %s has no named error result, skipping\n",
						fset.Position(comment.Pos()), ctx.funcName,
					)
					continue
				}

				ctx.errsName = errsNameAt(comment.Pos())
				replacement := generateWrapStatement(ctx)
				if verboseOut != nil {
					fmt.Fprintf(verboseOut, "%s: replacing //#wrap-result-err with %s\n", fset.Position(comment.Pos()), replacement)
				}
				replacements.AddReplacement(comment, replacement, "replace //#wrap-result-err")
			}
		}
	}

	// For insert mode, find functions with named error results that don't have defer errs.Wrap
	if mode == modeInsert {
		ast.Inspect(astFile, func(n ast.Node) bool {
			if n == nil {
				return true
			}

			var funcType *ast.FuncType
			var funcBody *ast.BlockStmt
			var funcName string
			var startPos, endPos token.Pos

			switch node := n.(type) {
			case *ast.FuncDecl:
				if node.Body == nil {
					return true // Skip function declarations without body
				}
				funcType = node.Type
				funcBody = node.Body
				funcName = node.Name.Name
				startPos = node.Pos()
				endPos = node.End()
			case *ast.FuncLit:
				funcType = node.Type
				funcBody = node.Body
				funcName = "(anonymous)"
				startPos = node.Pos()
				endPos = node.End()
			default:
				return true
			}

			// Skip if already has defer errs.Wrap
			if funcsWithWrap[startPos] {
				return true
			}

			// Extract function info
			fun := extractFuncInfo(funcType, funcName, startPos, endPos)
			if fun.errorResultName == "" {
				return true // Skip functions without named error result
			}

			// Find the position to insert (after the opening brace of the function body)
			fun.errsName = errsNameAt(funcBody.Lbrace)
			statement := generateWrapStatement(fun)
			if len(funcBody.List) == 0 {
				// Empty function body - insert after opening brace
				insertPos := funcBody.Lbrace + 1
				if verboseOut != nil {
					fmt.Fprintf(verboseOut, "%s: inserting %s\n", fset.Position(insertPos), statement)
				}
				// Use PosNode to create a zero-width insertion point
				replacements.AddReplacement(astvisit.PosNode(insertPos), "\n\t"+statement+"\n", "insert defer errs.Wrap")
			} else {
				// Insert before the first statement, with empty line after
				firstStmt := funcBody.List[0]
				if verboseOut != nil {
					fmt.Fprintf(verboseOut, "%s: inserting %s\n", fset.Position(firstStmt.Pos()), statement)
				}
				// Use PosNode to create a zero-width insertion point before the first statement
				replacements.AddReplacement(astvisit.PosNode(firstStmt.Pos()), statement+"\n\n\t", "insert defer errs.Wrap")
			}

			return true
		})
	}

	if len(replacements) == 0 {
		return nil, nil, nil
	}
	for name := range addedErrsNames {
		if name == "errs" {
			imports.Add("github.com/domonda/go-errs")
		} else {
			imports.AddNamed(name, "github.com/domonda/go-errs")
		}
	}

	return replacements, imports, nil
}

// isDeferErrsWrap checks if a defer statement is calling errs.Wrap* function
// using any of the known import aliases for go-errs.
func isDeferErrsWrap(stmt *ast.DeferStmt, errsAliases map[string]bool) bool {
	call, ok := stmt.Call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	ident, ok := call.X.(*ast.Ident)
	if !ok {
		return false
	}

	// An identifier resolved by the parser is a declaration
	// of the file, like a parameter, that shadows the package name
	if !errsAliases[ident.Name] || ident.Obj != nil {
		return false
	}

	return strings.HasPrefix(call.Sel.Name, "Wrap")
}

// isVariadicWrapWithFuncParams checks if a defer statement is calling
// the variadic errs.WrapWithFuncParams function (not a specialized version)
// using any of the known import aliases for go-errs.
func isVariadicWrapWithFuncParams(stmt *ast.DeferStmt, errsAliases map[string]bool) bool {
	call, ok := stmt.Call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	ident, ok := call.X.(*ast.Ident)
	if !ok {
		return false
	}

	// An identifier resolved by the parser is a declaration
	// of the file, like a parameter, that shadows the package name
	if !errsAliases[ident.Name] || ident.Obj != nil {
		return false
	}

	return call.Sel.Name == "WrapWithFuncParams"
}

// replacementChangesSource reports whether applying repl would actually
// change the source text at the node's position. Used in validate mode to
// avoid false positives from whole-file comparisons where unrelated
// formatting (e.g. import reordering) makes the rewritten file differ
// even though every replacement is a no-op.
func replacementChangesSource(fset *token.FileSet, source []byte, repl astvisit.NodeReplacement) bool {
	start := fset.Position(repl.Node.Pos()).Offset
	end := fset.Position(repl.Node.End()).Offset
	if start < 0 || end < start || end > len(source) {
		// Can't determine original slice - treat as a change to be safe.
		return true
	}
	original := source[start:end]

	switch r := repl.Replacement.(type) {
	case nil:
		// Removal always changes the source.
		return true
	case string:
		return string(original) != r
	case []byte:
		return !bytes.Equal(original, r)
	default:
		// Unknown replacement types may format non-deterministically; assume change.
		return true
	}
}

// copyFile copies the file src to dst with perm, see writeFile.
func copyFile(src, dst string, perm os.FileMode) error {
	// #nosec G304 -- src is a file below the source directory found by filepath.Walk
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	return writeFile(dst, source, perm)
}

// writeFile writes the content of r with perm to a new file that replaces
// path, so that a hard or symbolic link at path is replaced instead of
// writing through it to the file it links to.
func writeFile(path string, r io.Reader, perm os.FileMode) (err error) {
	file, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*")
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = os.Remove(file.Name())
		}
	}()

	_, err = io.Copy(file, r)
	if err == nil {
		err = file.Chmod(perm)
	}
	err = errors.Join(err, file.Close())
	if err != nil {
		return err
	}
	return os.Rename(file.Name(), path)
}
