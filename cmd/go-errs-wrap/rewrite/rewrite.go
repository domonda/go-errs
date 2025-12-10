// Package rewrite provides functionality to remove or replace
// defer errs.Wrap statements and //#wrap-result-err marker comments
// in Go source files.
package rewrite

import (
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

// Remove removes all defer errs.Wrap statements and //#wrap-result-err
// marker comments from Go source files at the given path.
//
// If outPath is empty, files are modified in place.
// If outPath is specified, results are written there instead.
// If recursive is true, subdirectories are processed recursively.
func Remove(sourcePath, outPath string, recursive bool, verboseOut io.Writer) (err error) {
	defer errs.WrapWithFuncParams(&err, sourcePath, outPath, recursive, verboseOut)

	return process(sourcePath, outPath, recursive, false, verboseOut, true)
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
func Replace(sourcePath, outPath string, recursive, minVariadic bool, verboseOut io.Writer) (err error) {
	defer errs.WrapWithFuncParams(&err, sourcePath, outPath, recursive, minVariadic, verboseOut)

	return process(sourcePath, outPath, recursive, minVariadic, verboseOut, false)
}

func process(sourcePath, outPath string, recursive, minVariadic bool, verboseOut io.Writer, removeOnly bool) (err error) {
	defer errs.WrapWithFuncParams(&err, sourcePath, outPath, recursive, minVariadic, verboseOut, removeOnly)

	sourcePath, err = filepath.Abs(sourcePath)
	if err != nil {
		return err
	}

	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		return err
	}

	// Handle output path for directory
	if outPath != "" {
		outPath, err = filepath.Abs(outPath)
		if err != nil {
			return err
		}

			if sourceInfo.IsDir() {
			return processDirectoryWithOutput(sourcePath, outPath, recursive, minVariadic, verboseOut, removeOnly)
		}

		// For single file with output, use custom handling
		return processSingleFileWithOutput(sourcePath, outPath, minVariadic, verboseOut, removeOnly)
	}

	// In-place modification
	// Use "..." suffix for directories to recursively find all packages
	path := sourcePath
	if sourceInfo.IsDir() && recursive {
		path = strings.TrimSuffix(sourcePath, "/") + "/..."
	}
	return astvisit.RewriteWithReplacements(
		path,
		verboseOut,
		nil, // modify in place
		false,
		func(fset *token.FileSet, _ *ast.Package, astFile *ast.File, filePath string, verboseOut io.Writer) (astvisit.NodeReplacements, astvisit.Imports, error) {
			return processFile(fset, astFile, minVariadic, verboseOut, removeOnly)
		},
	)
}

func processSingleFileWithOutput(sourcePath, outPath string, minVariadic bool, verboseOut io.Writer, removeOnly bool) (err error) {
	defer errs.WrapWithFuncParams(&err, sourcePath, outPath, minVariadic, verboseOut, removeOnly)

	// Determine final output path
	outInfo, err := os.Stat(outPath)
	if err == nil && outInfo.IsDir() {
		// Output is existing directory, add source filename
		outPath = filepath.Join(outPath, filepath.Base(sourcePath))
	} else if !strings.HasSuffix(outPath, ".go") {
		// If outPath doesn't exist and doesn't end with .go,
		// treat it as a directory to create
		if err := os.MkdirAll(outPath, 0755); err != nil {
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
				replacements, imports, err := processFile(fset, astFile, minVariadic, verboseOut, removeOnly)
			if err != nil {
				return nil, nil, err
			}

			// Read original source
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

				if removeOnly {
					// For remove, use goimports to remove unused imports
					rewritten, err = goimports.Process(filePath, rewritten, nil)
					if err != nil {
						return nil, nil, err
					}
				} else {
					// For replace, format with imports to ensure errs is imported
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
			if err := os.WriteFile(outPath, rewritten, 0644); err != nil {
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

func processDirectoryWithOutput(sourcePath, outPath string, recursive, minVariadic bool, verboseOut io.Writer, removeOnly bool) (err error) {
	defer errs.WrapWithFuncParams(&err, sourcePath, outPath, recursive, minVariadic, verboseOut, removeOnly)

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
				replacements, imports, err := processFile(fset, astFile, minVariadic, verboseOut, removeOnly)
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
				if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
					return nil, nil, err
				}

				// Read original file
				source, err := os.ReadFile(filePath)
				if err != nil {
					return nil, nil, err
				}

				// Apply replacements
				rewritten, err := replacements.Apply(fset, source)
				if err != nil {
					return nil, nil, err
				}

				if removeOnly {
					// For remove, use goimports to remove unused imports
					rewritten, err = goimports.Process(filePath, rewritten, nil)
					if err != nil {
						return nil, nil, err
					}
				} else {
					// For replace, format with imports to ensure errs is imported
					rewritten, err = astvisit.FormatFileWithImports(fset, rewritten, imports)
					if err != nil {
						return nil, nil, err
					}
				}

				// Write to destination
				if err := os.WriteFile(destPath, rewritten, 0644); err != nil {
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

func processFile(fset *token.FileSet, astFile *ast.File, minVariadic bool, verboseOut io.Writer, removeOnly bool) (replacements astvisit.NodeReplacements, imports astvisit.Imports, err error) {
	defer errs.WrapWithFuncParams(&err, fset, astFile, minVariadic, verboseOut, removeOnly)

	imports = make(astvisit.Imports)

	// Visit all nodes to find defer statements
	ast.Inspect(astFile, func(n ast.Node) bool {
		if n == nil {
			return true
		}

		deferStmt, ok := n.(*ast.DeferStmt)
		if !ok || !isDeferErrsWrap(deferStmt) {
			return true
		}

		if removeOnly {
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

		// If already using variadic WrapWithFuncParams and minVariadic is false, keep it variadic
		var replacement string
		if !minVariadic && isVariadicWrapWithFuncParams(deferStmt) {
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

	// Process comments for //#wrap-result-err markers
	for _, commentGroup := range astFile.Comments {
		for _, comment := range commentGroup.List {
			if !strings.HasPrefix(comment.Text, "//") {
				continue
			}
			trimmed := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
			if trimmed != "#wrap-result-err" {
				continue
			}

			if removeOnly {
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

	if len(replacements) == 0 {
		return nil, nil, nil
	}

	return replacements, imports, nil
}

// isDeferErrsWrap checks if a defer statement is calling errs.Wrap* function
func isDeferErrsWrap(stmt *ast.DeferStmt) bool {
	call, ok := stmt.Call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	ident, ok := call.X.(*ast.Ident)
	if !ok {
		return false
	}

	if ident.Name != "errs" {
		return false
	}

	return strings.HasPrefix(call.Sel.Name, "Wrap")
}

// isVariadicWrapWithFuncParams checks if a defer statement is calling
// the variadic errs.WrapWithFuncParams function (not a specialized version).
func isVariadicWrapWithFuncParams(stmt *ast.DeferStmt) bool {
	call, ok := stmt.Call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}

	ident, ok := call.X.(*ast.Ident)
	if !ok {
		return false
	}

	if ident.Name != "errs" {
		return false
	}

	return call.Sel.Name == "WrapWithFuncParams"
}

func copyFile(src, dst string) error {
	source, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.WriteFile(dst, source, info.Mode())
}
