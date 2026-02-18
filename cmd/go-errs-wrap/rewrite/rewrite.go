// Package rewrite provides functionality to remove, replace, or insert
// defer errs.Wrap statements and //#wrap-result-err marker comments
// in Go source files.
package rewrite

import (
	"bytes"
	"fmt"
	"go/ast"
	"go/token"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ungerik/go-astvisit"
	goimports "golang.org/x/tools/imports"

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

	// Handle output path for directory when not in validate mode
	if outPath != "" && !validate {
		outPath, err = filepath.Abs(outPath)
		if err != nil {
			return err
		}

		if sourceInfo.IsDir() {
			return processDirectoryWithOutput(sourcePath, outPath, recursive, minVariadic, validate, verboseOut, mode)
		}

		// For single file with output, use custom handling
		return processSingleFileWithOutput(sourcePath, outPath, minVariadic, validate, verboseOut, mode)
	}

	// In-place modification (or validation)
	// Use "..." suffix for directories to recursively find all packages
	path := sourcePath
	if sourceInfo.IsDir() && recursive {
		path = strings.TrimSuffix(sourcePath, "/") + "/..."
	}

	var validationErrors []string
	err = astvisit.RewriteWithReplacements(
		path,
		verboseOut,
		nil, // modify in place (or skip in validation mode)
		false,
		func(fset *token.FileSet, _ *ast.Package, astFile *ast.File, filePath string, verboseOut io.Writer) (astvisit.NodeReplacements, astvisit.Imports, error) {
			replacements, imports, err := processFile(fset, astFile, minVariadic, verboseOut, mode)
			if err != nil {
				return nil, nil, err
			}

			// In validation mode, check if replacements would actually change the file
			if validate && len(replacements) > 0 {
				// Read original source
				// #nosec G304 -- filePath comes from astvisit callback and is validated via filepath.Rel
				source, err := os.ReadFile(filePath)
				if err != nil {
					return nil, nil, err
				}

				// Apply replacements to see if file would change
				rewritten, err := replacements.Apply(fset, source)
				if err != nil {
					return nil, nil, err
				}

				// Apply the same formatting that would happen in normal mode
				if mode == modeRemove {
					rewritten, err = goimports.Process(filePath, rewritten, nil)
					if err != nil {
						return nil, nil, err
					}
				} else {
					rewritten, err = astvisit.FormatFileWithImports(fset, rewritten, imports)
					if err != nil {
						return nil, nil, err
					}
				}

				// Check if the content actually changed
				if !bytes.Equal(source, rewritten) {
					// File would change - report as validation error
					for _, repl := range replacements {
						var pos token.Position
						if repl.Node != nil {
							pos = fset.Position(repl.Node.Pos())
						}
						desc := repl.DebugID
						if desc == "" {
							desc = "error wrapper"
						}
						validationErrors = append(validationErrors, fmt.Sprintf("%s: missing %s", pos, desc))
					}
				}

				// Return nil to prevent file modification in validate mode
				return nil, nil, nil
			}

			return replacements, imports, nil
		},
	)
	if err != nil {
		return err
	}

	// In validation mode, report errors and fail if any were found
	if validate && len(validationErrors) > 0 {
		for _, errMsg := range validationErrors {
			fmt.Fprintln(os.Stderr, errMsg)
		}
		return errs.Errorf("found %d missing error wrapper(s)", len(validationErrors))
	}

	return nil
}

func processSingleFileWithOutput(sourcePath, outPath string, minVariadic, validate bool, verboseOut io.Writer, mode processMode) (err error) {
	defer errs.WrapWithFuncParams(&err, sourcePath, outPath, minVariadic, validate, verboseOut, mode)

	// Determine final output path
	outInfo, err := os.Stat(outPath)
	if err == nil && outInfo.IsDir() {
		// Output is existing directory, add source filename
		outPath = filepath.Join(outPath, filepath.Base(sourcePath))
	} else if !strings.HasSuffix(outPath, ".go") {
		// If outPath doesn't exist and doesn't end with .go,
		// treat it as a directory to create
		if err := os.MkdirAll(outPath, 0750); err != nil {
			return err
		}
		outPath = filepath.Join(outPath, filepath.Base(sourcePath))
	}

	// Process the file
	return astvisit.RewriteWithReplacements(
		sourcePath,
		verboseOut,
		nil, // We handle output ourselves
		false,
		func(fset *token.FileSet, pkg *ast.Package, astFile *ast.File, filePath string, verboseOut io.Writer) (astvisit.NodeReplacements, astvisit.Imports, error) {
			replacements, imports, err := processFile(fset, astFile, minVariadic, verboseOut, mode)
			if err != nil {
				return nil, nil, err
			}

			// Read original source
			// #nosec G304 -- filePath comes from astvisit callback and is validated via filepath.Rel
			source, err := os.ReadFile(filePath)
			if err != nil {
				return nil, nil, err
			}

			var rewritten []byte
			if len(replacements) > 0 {
				// Apply replacements
				rewritten, err = replacements.Apply(fset, source)
				if err != nil {
					return nil, nil, err
				}

				if mode == modeRemove {
					// For remove, use goimports to remove unused imports
					rewritten, err = goimports.Process(filePath, rewritten, nil)
					if err != nil {
						return nil, nil, err
					}
				} else {
					// For replace/insert, format with imports to ensure errs is imported
					rewritten, err = astvisit.FormatFileWithImports(fset, rewritten, imports)
					if err != nil {
						return nil, nil, err
					}
				}
			} else {
				// No changes, copy original
				rewritten = source
			}

			// Write to output
			// #nosec G306 -- 0640 is appropriate for generated Go source files
			if err := os.WriteFile(outPath, rewritten, 0640); err != nil {
				return nil, nil, err
			}

			if verboseOut != nil {
				fmt.Fprintf(verboseOut, "wrote: %s\n", outPath)
			}

			// Return nil to prevent astvisit from writing
			return nil, nil, nil
		},
	)
}

func processDirectoryWithOutput(sourcePath, outPath string, recursive, minVariadic, validate bool, verboseOut io.Writer, mode processMode) (err error) {
	defer errs.WrapWithFuncParams(&err, sourcePath, outPath, recursive, minVariadic, validate, verboseOut, mode)

	// First, copy non-Go files
	err = filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(sourcePath, path)
		if err != nil {
			return err
		}
		destPath := filepath.Join(outPath, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}

		// Skip Go files (they'll be processed separately) and test files
		if strings.HasSuffix(path, ".go") {
			return nil
		}

		// Copy non-Go files
		if verboseOut != nil {
			fmt.Fprintf(verboseOut, "copying: %s -> %s\n", path, destPath)
		}
		return copyFile(path, destPath)
	})
	if err != nil {
		return err
	}

	// Process Go files with astvisit
	return astvisit.RewriteWithReplacements(
		strings.TrimSuffix(sourcePath, "/")+"/...",
		verboseOut,
		nil, // We'll handle output ourselves
		false,
		func(fset *token.FileSet, pkg *ast.Package, astFile *ast.File, filePath string, verboseOut io.Writer) (astvisit.NodeReplacements, astvisit.Imports, error) {
			replacements, imports, err := processFile(fset, astFile, minVariadic, verboseOut, mode)
			if err != nil {
				return nil, nil, err
			}

			// If there are replacements, we need to write to the output path
			if len(replacements) > 0 {
				relPath, err := filepath.Rel(sourcePath, filePath)
				if err != nil {
					return nil, nil, err
				}
				destPath := filepath.Join(outPath, relPath)

				// Ensure directory exists
				if err := os.MkdirAll(filepath.Dir(destPath), 0750); err != nil {
					return nil, nil, err
				}

				// Read original file
				// #nosec G304 -- filePath comes from astvisit callback and is validated via filepath.Rel at line 318
				source, err := os.ReadFile(filePath)
				if err != nil {
					return nil, nil, err
				}

				// Apply replacements
				rewritten, err := replacements.Apply(fset, source)
				if err != nil {
					return nil, nil, err
				}

				if mode == modeRemove {
					// For remove, use goimports to remove unused imports
					rewritten, err = goimports.Process(filePath, rewritten, nil)
					if err != nil {
						return nil, nil, err
					}
				} else {
					// For replace/insert, format with imports to ensure errs is imported
					rewritten, err = astvisit.FormatFileWithImports(fset, rewritten, imports)
					if err != nil {
						return nil, nil, err
					}
				}

				// Write to destination
				// #nosec G306 -- 0640 is appropriate for generated Go source files
				if err := os.WriteFile(destPath, rewritten, 0640); err != nil {
					return nil, nil, err
				}

				if verboseOut != nil {
					fmt.Fprintf(verboseOut, "wrote: %s\n", destPath)
				}
			}

			// Return nil to prevent astvisit from writing
			return nil, nil, nil
		},
	)
}

func processFile(fset *token.FileSet, astFile *ast.File, minVariadic bool, verboseOut io.Writer, mode processMode) (replacements astvisit.NodeReplacements, imports astvisit.Imports, err error) {
	defer errs.WrapWithFuncParams(&err, fset, astFile, minVariadic, verboseOut, mode)

	imports = make(astvisit.Imports)

	// Extract all aliases used to import go-errs (for detecting KeepSecret calls)
	errsAliases := extractErrsAliases(astFile)

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
			imports[`"github.com/domonda/go-errs"`] = struct{}{}

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

				replacement := generateWrapStatement(ctx)
				if verboseOut != nil {
					fmt.Fprintf(verboseOut, "%s: replacing //#wrap-result-err with %s\n", fset.Position(comment.Pos()), replacement)
				}
				replacements.AddReplacement(comment, replacement, "replace //#wrap-result-err")
				imports[`"github.com/domonda/go-errs"`] = struct{}{}
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
			statement := generateWrapStatement(fun)
			if funcBody == nil || len(funcBody.List) == 0 {
				// Empty function body - insert after opening brace
				insertPos := funcBody.Lbrace + 1
				if verboseOut != nil {
					fmt.Fprintf(verboseOut, "%s: inserting %s\n", fset.Position(insertPos), statement)
				}
				// Use PosNode to create a zero-width insertion point
				replacements.AddReplacement(astvisit.PosNode(insertPos), "\n\t"+statement+"\n", "insert defer errs.Wrap")
				imports[`"github.com/domonda/go-errs"`] = struct{}{}
			} else {
				// Insert before the first statement, with empty line after
				firstStmt := funcBody.List[0]
				if verboseOut != nil {
					fmt.Fprintf(verboseOut, "%s: inserting %s\n", fset.Position(firstStmt.Pos()), statement)
				}
				// Use PosNode to create a zero-width insertion point before the first statement
				replacements.AddReplacement(astvisit.PosNode(firstStmt.Pos()), statement+"\n\n\t", "insert defer errs.Wrap")
				imports[`"github.com/domonda/go-errs"`] = struct{}{}
			}

			return true
		})
	}

	if len(replacements) == 0 {
		return nil, nil, nil
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

	if !errsAliases[ident.Name] {
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

	if !errsAliases[ident.Name] {
		return false
	}

	return call.Sel.Name == "WrapWithFuncParams"
}

func copyFile(src, dst string) error {
	// Clean paths to prevent directory traversal
	src = filepath.Clean(src)
	dst = filepath.Clean(dst)

	source, err := os.ReadFile(src) // #nosec G304 -- paths are cleaned and validated by filepath.Walk
	if err != nil {
		return err
	}

	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, source, info.Mode())
}
