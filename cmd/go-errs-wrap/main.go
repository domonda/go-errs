/*
go-errs-wrap is a code transformation tool for managing error wrapping in Go.

It scans Go source files for defer errs.Wrap statements or //#wrap-result-err
marker comments and can remove them, replace them with properly generated
defer errs.WrapWithFuncParams statements, or insert new wrap statements into
functions that don't have them yet.

# Usage

	go-errs-wrap <command> [options] <path>

# Commands

	remove   Remove all defer errs.Wrap or //#wrap-result-err lines
	replace  Replace defer errs.Wrap or //#wrap-result-err with generated code
	insert   Insert defer errs.Wrap at the first line of functions with named
	         error results that don't already have one (followed by empty line)

# Options

	-out <path>   Output to different location instead of modifying source
	-validate     Dry run mode: check for issues without modifying files. Reports issues
	              to stderr and exits with error code 1 if any are found. Useful for CI
	              validation to ensure code quality standards are met.
	              - remove: checks if any defer errs.Wrap statements exist
	              - replace: checks if any defer errs.Wrap statements need updating
	              - insert: checks if any functions are missing defer errs.Wrap
	-verbose      Print progress information
	-help         Show help message

# Exit Codes

	0   Success (no issues found in validate mode, or operation completed successfully)
	1   Error occurred (validation failed, file not found, invalid arguments, etc.)

# Examples

Remove all wrap statements from a directory:

	go-errs-wrap remove ./pkg/...

Replace wrap statements in a single file:

	go-errs-wrap replace ./pkg/mypackage/file.go

Insert wrap statements into functions missing them:

	go-errs-wrap insert ./pkg/mypackage/file.go

Replace and output to a different location:

	go-errs-wrap replace -out ./output ./pkg/mypackage

Validate that no defer errs.Wrap statements exist (for CI):

	go-errs-wrap remove -validate ./pkg/...

Validate that all replacements are present (for CI):

	go-errs-wrap replace -validate ./pkg/...

Validate that all functions have error wrappers (for CI):

	go-errs-wrap insert -validate ./pkg/...
*/
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/domonda/go-errs"
	"github.com/domonda/go-errs/cmd/go-errs-wrap/rewrite"
)

var (
	outPath     string
	verbose     bool
	minVariadic bool
	validate    bool
	printHelp   bool
)

func main() {
	// Check for help first
	if len(os.Args) > 1 && (os.Args[1] == "-help" || os.Args[1] == "--help" || os.Args[1] == "-h") {
		printUsage()
		os.Exit(0)
	}

	// Need at least command and path
	if len(os.Args) < 3 {
		fmt.Fprintln(os.Stderr, "error: missing command and/or path argument")
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	// Create a new FlagSet for parsing flags after the command
	fs := flag.NewFlagSet(command, flag.ExitOnError)
	fs.StringVar(&outPath, "out", "", "output to different location instead of modifying source")
	fs.BoolVar(&verbose, "verbose", false, "print progress information")
	fs.BoolVar(&minVariadic, "minvariadic", false, "minimize use of variadic WrapWithFuncParams")
	fs.BoolVar(&validate, "validate", false, "check for issues without modifying files")
	fs.BoolVar(&printHelp, "help", false, "show help message")
	fs.Parse(os.Args[2:]) // #nosec G104 -- using ExitOnError mode, Parse will exit on error

	if printHelp {
		printUsage()
		os.Exit(0)
	}

	args := fs.Args()
	if len(args) < 1 {
		fmt.Fprintln(os.Stderr, "error: missing path argument")
		printUsage()
		os.Exit(1)
	}

	sourcePath := args[0]

	// Check if user passed a "..." pattern (e.g., "./..." or "./pkg/...")
	recursive := false
	if strings.HasSuffix(sourcePath, "/...") {
		sourcePath = strings.TrimSuffix(sourcePath, "/...")
		recursive = true
	} else if sourcePath == "..." {
		sourcePath = "."
		recursive = true
	}

	var verboseOut io.Writer
	if verbose {
		verboseOut = os.Stdout
	}

	var err error
	switch command {
	case "remove":
		err = rewrite.Remove(sourcePath, outPath, recursive, validate, verboseOut)
	case "replace":
		err = rewrite.Replace(sourcePath, outPath, recursive, minVariadic, validate, verboseOut)
	case "insert":
		err = rewrite.Insert(sourcePath, outPath, recursive, minVariadic, validate, verboseOut)
	default:
		fmt.Fprintf(os.Stderr, "error: unknown command %q\n", command) // #nosec G705 -- CLI stderr output, not HTTP response
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", errs.UnwrapCallStack(err))
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`go-errs-wrap - manage defer errs.WrapWithFuncParams statements

Usage:
  go-errs-wrap <command> [options] <path>

Commands:
  remove   Remove all defer errs.Wrap or //#wrap-result-err lines
  replace  Replace defer errs.Wrap or //#wrap-result-err with generated code
  insert   Insert defer errs.Wrap at the first line of functions with named
           error results that don't already have one (followed by empty line)

Arguments:
  path     Source directory or file path
           - If directory: recursively process all *.go files (excluding *_test.go)
           - If file: process only that file

Options:
  -out <path>     Output to different location instead of modifying source
                  - If source is directory: create copy of directory structure
                  - If source is file: write to specified file path
                  - Non-Go files are copied unchanged
  -validate       Dry run mode: check for issues without modifying files. Reports issues
                  to stderr and exits with error code 1 if any are found. Useful for CI
                  validation to ensure code quality standards are met.
                  - remove: checks if any defer errs.Wrap statements exist
                  - replace: checks if any defer errs.Wrap statements need updating
                  - insert: checks if any functions are missing defer errs.Wrap
                  Note: -out option is ignored when -validate is used.
  -minvariadic    Use specialized WrapWithNFuncParams functions instead of
                  preserving existing variadic WrapWithFuncParams calls
  -verbose        Print progress information
  -help           Show help message

Exit Codes:
  0   Success (no issues found in validate mode, or operation completed successfully)
  1   Error occurred (validation failed, file not found, invalid arguments, etc.)

Examples:
  go-errs-wrap remove ./pkg/...
  go-errs-wrap replace ./pkg/mypackage/file.go
  go-errs-wrap insert ./pkg/mypackage/file.go
  go-errs-wrap replace -out ./output ./pkg/mypackage
  go-errs-wrap remove -validate ./pkg/...
  go-errs-wrap replace -validate ./pkg/...
  go-errs-wrap insert -validate ./pkg/...`)
}
