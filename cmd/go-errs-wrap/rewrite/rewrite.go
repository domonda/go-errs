// Package rewrite provides functionality to remove, replace, or insert
// defer errs.Wrap statements and //#wrap-result-err marker comments
// in Go source files.
package rewrite

import (
	"bytes"
	"crypto/rand"
	"errors"
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"

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
	} else if isSameFile(sourcePath, singleFileOutputPath(sourcePath, outPath)) {
		// The output path is the source file or its directory
		return errs.Errorf("output path %s must not be the source file %s or its directory", outPath, sourcePath)
	}

	// Rewrite all files before writing any output, so that an error
	// of the rewrite doesn't leave a partially written output,
	// errors while writing the output still can
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
		// Only needed for files without a go-errs import or with
		// a new go-errs import name, loaded once for all of them
		loadPackageNames := sync.OnceValues(func() (map[string]bool, error) {
			return packageNames(pkg)
		})
		rewritten := make(map[string][]byte)
		for _, filePath := range pkg.FilePaths {
			source := pkg.Sources[filePath]
			replacements, imports, err := processFile(pkg.Fset, pkg.Files[filePath], loadPackageNames, minVariadic, verboseOut, mode)
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
// and formatted like gofmt. For replace and insert the missing imports
// are added, and a go-errs import that is not used anymore is removed.
func rewriteSource(fset *token.FileSet, filePath string, source []byte, replacements astvisit.NodeReplacements, imports astvisit.Imports, mode processMode) ([]byte, error) {
	rewritten, err := replacements.Apply(fset, source)
	if err != nil {
		return nil, err
	}
	if mode != modeRemove {
		rewritten, err = astvisit.FormatSource(rewritten, imports)
		if err != nil {
			return nil, err
		}
	}
	return removeUnusedErrsImports(filePath, rewritten)
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
// or its unchanged source if rewritten is nil, to outPath
// with the permissions of sourceFile, see writeFile.
// If outPath is an existing directory or doesn't end with .go,
// the file is written with the name of sourceFile into that directory,
// which is created if missing.
func writeFileOutput(sourceFile, outPath string, rewritten []byte, verboseOut io.Writer) error {
	outFile := singleFileOutputPath(sourceFile, outPath)
	if outFile != outPath {
		err := os.MkdirAll(outPath, 0750)
		if err != nil {
			return err
		}
	}
	outPath = outFile

	sourceInfo, err := os.Stat(sourceFile)
	if err != nil {
		return err
	}
	if rewritten == nil {
		// #nosec G304 -- sourceFile is the file path passed by the user
		rewritten, err = os.ReadFile(sourceFile)
		if err != nil {
			return err
		}
	}

	root, err := os.OpenRoot(filepath.Dir(outPath))
	if err != nil {
		return err
	}
	defer root.Close()
	err = writeFile(root, filepath.Base(outPath), bytes.NewReader(rewritten), sourceInfo.Mode().Perm()|0200)
	if err != nil {
		return err
	}
	if verboseOut != nil {
		fmt.Fprintf(verboseOut, "wrote: %s\n", outPath)
	}
	return nil
}

// singleFileOutputPath returns the path that the source file sourceFile
// is written to for the output path outPath: outPath itself if it ends with .go
// and is not an existing directory, else a file named like sourceFile
// in the directory outPath.
func singleFileOutputPath(sourceFile, outPath string) string {
	outInfo, err := os.Stat(outPath)
	isDir := err == nil && outInfo.IsDir()
	if !isDir && strings.HasSuffix(outPath, ".go") {
		return outPath
	}
	return filepath.Join(outPath, filepath.Base(sourceFile))
}

// isSameFile reports whether path1 and path2 are the same existing
// directory entry, also when spelled differently, like another letter case
// on a case-insensitive file system or a directory path through a link.
// A link is not the same as the file it links to, because writeFile
// replaces a link instead of writing through it.
func isSameFile(path1, path2 string) bool {
	dirInfo1, err := os.Stat(filepath.Dir(path1))
	if err != nil {
		return false
	}
	dirInfo2, err := os.Stat(filepath.Dir(path2))
	if err != nil {
		return false
	}
	info1, err := os.Lstat(path1)
	if err != nil {
		return false
	}
	info2, err := os.Lstat(path2)
	if err != nil {
		return false
	}
	return os.SameFile(dirInfo1, dirInfo2) && os.SameFile(info1, info2)
}

// writeDirectoryOutput copies the files below sourceDir, or without recursive
// only the files directly in sourceDir, to outDir and writes the rewritten
// sources instead of the original ones of the files in rewritten.
// Version control metadata and files that are not regular files
// or symbolic links to regular files are not copied.
// All files are written within outDir, symbolic links in outDir
// that lead outside of it are an error, see writeFile.
// Copied files and directories are writable by the owner,
// so that the output can be written again.
func writeDirectoryOutput(sourceDir, outDir string, recursive bool, rewritten map[string][]byte, verboseOut io.Writer) error {
	// filepath.Walk doesn't follow a symbolic link as root
	sourceRoot, err := filepath.EvalSymlinks(sourceDir)
	if err != nil {
		return err
	}
	sourceRootInfo, err := os.Stat(sourceRoot)
	if err != nil {
		return err
	}
	err = os.MkdirAll(outDir, sourceRootInfo.Mode().Perm()|0700)
	if err != nil {
		return err
	}
	outRoot, err := os.OpenRoot(outDir)
	if err != nil {
		return err
	}
	defer outRoot.Close()

	return filepath.Walk(sourceRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(sourceRoot, path)
		if err != nil {
			return err
		}
		// astvisit names the files with the unresolved source directory
		sourcePath := filepath.Join(sourceDir, relPath)
		destPath := filepath.Join(outDir, relPath)

		if info.IsDir() {
			if path == sourceRoot {
				return nil
			}
			if !recursive || isVCSMetadata(info.Name()) {
				return filepath.SkipDir
			}
			return outRoot.MkdirAll(relPath, info.Mode().Perm()|0700)
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
			return writeFile(outRoot, relPath, bytes.NewReader(source), info.Mode().Perm()|0200)
		}
		if verboseOut != nil {
			fmt.Fprintf(verboseOut, "copying: %s -> %s\n", sourcePath, destPath)
		}
		return copyFile(path, outRoot, relPath, info.Mode().Perm()|0200)
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

// newStatement is a defer errs.Wrap statement that processFile generates
// for a //#wrap-result-err marker or inserts into a function.
type newStatement struct {
	fun     *funcInfo
	pos     token.Pos // where the name referencing go-errs must not be shadowed
	node    ast.Node  // replaced by the statement
	before  string    // before the statement in the replacement
	after   string    // after the statement in the replacement
	verb    string    // describes the replacement in verbose output
	debugID string
}

func processFile(fset *token.FileSet, astFile *ast.File, loadPackageNames func() (map[string]bool, error), minVariadic bool, verboseOut io.Writer, mode processMode) (replacements astvisit.NodeReplacements, imports astvisit.Imports, err error) {
	defer errs.WrapWithFuncParams(&err, fset, astFile, loadPackageNames, minVariadic, verboseOut, mode)

	errsNames, importsErrs := errsImportNames(astFile)
	// isReserved reports whether go-errs can't be imported under name,
	// because another import or a package level declaration has it.
	// The package level names are only loaded for files that need them
	otherNames := otherImportNames(astFile)
	var packageNames map[string]bool
	isReserved := func(name string) bool {
		return otherNames[name] || packageNames[name]
	}

	// The names that reference go-errs in the file (for detecting its statements):
	// the names of its go-errs imports, or errs for a file without such an import
	errsAliases := make(map[string]bool)
	for _, name := range errsNames {
		errsAliases[name] = true
	}
	if !importsErrs {
		packageNames, err = loadPackageNames()
		if err != nil {
			return nil, nil, err
		}
		if !isReserved("errs") {
			errsAliases["errs"] = true
		}
	}

	// The names that generated statements reference go-errs by
	usedNames := make(map[string]bool)
	// Statements for markers and insertions, generated after all are known
	var newStatements []newStatement

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
			// Keep the name that the statement references go-errs by,
			// so that a statement that is up to date stays unchanged
			fun.errsName, _ = errsSelector(deferStmt.Call.Fun, errsAliases)
			usedNames[fun.errsName] = true

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

				newStatements = append(newStatements, newStatement{
					fun:     ctx,
					pos:     comment.Pos(),
					node:    comment,
					verb:    "replacing //#wrap-result-err with",
					debugID: "replace //#wrap-result-err",
				})
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

			// Insert after the opening brace of the function body,
			// where no declaration of the body is in scope yet
			stmt := newStatement{
				fun:     fun,
				pos:     funcBody.Lbrace,
				verb:    "inserting",
				debugID: "insert defer errs.Wrap",
			}
			if len(funcBody.List) == 0 {
				// Empty function body - insert after opening brace,
				// PosNode is a zero-width insertion point
				stmt.node = astvisit.PosNode(funcBody.Lbrace + 1)
				stmt.before = "\n\t"
				stmt.after = "\n"
			} else {
				// Insert before the first statement, with empty line after
				stmt.node = astvisit.PosNode(funcBody.List[0].Pos())
				stmt.after = "\n\n\t"
			}
			newStatements = append(newStatements, stmt)

			return true
		})
	}

	// New statements reference go-errs by the first name of its imports that
	// is not shadowed at their position. All others share one new import name,
	// so that go-errs is imported at most once more
	var fallbackPositions []token.Pos
	for _, stmt := range newStatements {
		stmt.fun.errsName = firstUnshadowedName(astFile, stmt.pos, errsNames)
		if stmt.fun.errsName == "" {
			fallbackPositions = append(fallbackPositions, stmt.pos)
		}
	}
	if len(fallbackPositions) > 0 {
		packageNames, err = loadPackageNames()
		if err != nil {
			return nil, nil, err
		}
		fallbackName := newErrsImportName(astFile, fallbackPositions, isReserved)
		for _, stmt := range newStatements {
			if stmt.fun.errsName == "" {
				stmt.fun.errsName = fallbackName
			}
		}
	}
	for _, stmt := range newStatements {
		statement := generateWrapStatement(stmt.fun)
		if verboseOut != nil {
			fmt.Fprintf(verboseOut, "%s: %s %s\n", fset.Position(stmt.node.Pos()), stmt.verb, statement)
		}
		replacements.AddReplacement(stmt.node, stmt.before+statement+stmt.after, stmt.debugID)
		usedNames[stmt.fun.errsName] = true
	}

	if len(replacements) == 0 {
		return nil, nil, nil
	}
	imports = make(astvisit.Imports)
	for name := range usedNames {
		switch {
		case slices.Contains(errsNames, name):
			// Imported under that name already
		case name == "errs":
			imports.Add("github.com/domonda/go-errs")
		default:
			imports.AddNamed(name, "github.com/domonda/go-errs")
		}
	}

	return replacements, imports, nil
}

// errsSelector returns the package name and the selected name of expr
// if it selects from the go-errs package by one of errsAliases, like errs.Wrap,
// else empty strings. An identifier that the parser resolved to a declaration
// of the file, like a parameter, shadows the package name.
func errsSelector(expr ast.Expr, errsAliases map[string]bool) (pkg, name string) {
	sel, ok := expr.(*ast.SelectorExpr)
	if !ok {
		return "", ""
	}
	ident, ok := sel.X.(*ast.Ident)
	if !ok || !errsAliases[ident.Name] || ident.Obj != nil {
		return "", ""
	}
	return ident.Name, sel.Sel.Name
}

// isDeferErrsWrap checks if a defer statement is calling errs.Wrap* function
// using any of the known import aliases for go-errs.
func isDeferErrsWrap(stmt *ast.DeferStmt, errsAliases map[string]bool) bool {
	_, name := errsSelector(stmt.Call.Fun, errsAliases)
	return strings.HasPrefix(name, "Wrap")
}

// isVariadicWrapWithFuncParams checks if a defer statement is calling
// the variadic errs.WrapWithFuncParams function (not a specialized version)
// using any of the known import aliases for go-errs.
func isVariadicWrapWithFuncParams(stmt *ast.DeferStmt, errsAliases map[string]bool) bool {
	_, name := errsSelector(stmt.Call.Fun, errsAliases)
	return name == "WrapWithFuncParams"
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

// copyFile copies the file src to name within root with perm, see writeFile.
func copyFile(src string, root *os.Root, name string, perm os.FileMode) error {
	// #nosec G304 -- src is a file below the source directory found by filepath.Walk
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	return writeFile(root, name, source, perm)
}

// writeFile writes the content of r with perm to a new file that replaces
// name within root, so that a hard or symbolic link at name is replaced
// instead of writing through it to the file it links to.
// Symbolic links of the directories of name must not lead outside of root.
func writeFile(root *os.Root, name string, r io.Reader, perm os.FileMode) (err error) {
	// Not derived from name, which could make it longer than a file name may be
	tempName := filepath.Join(filepath.Dir(name), ".go-errs-wrap-"+rand.Text())
	file, err := root.OpenFile(tempName, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			_ = root.Remove(tempName)
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
	return root.Rename(tempName, name)
}
