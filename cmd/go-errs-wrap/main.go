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
	-verbose      Print progress information
	-help         Show help message

# Examples

Remove all wrap statements from a directory:

	go-errs-wrap remove ./pkg/...

Replace wrap statements in a single file:

	go-errs-wrap replace ./pkg/mypackage/file.go

Insert wrap statements into functions missing them:

	go-errs-wrap insert ./pkg/mypackage/file.go

Replace and output to a different location:

	go-errs-wrap replace -out ./output ./pkg/mypackage
*/
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/domonda/go-errs/cmd/go-errs-wrap/rewrite"
)

var (
	outPath     string
	verbose     bool
	minVariadic bool
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
	fs.BoolVar(&printHelp, "help", false, "show help message")
	fs.Parse(os.Args[2:])

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
		err = rewrite.Remove(sourcePath, outPath, recursive, verboseOut)
	case "replace":
		err = rewrite.Replace(sourcePath, outPath, recursive, minVariadic, verboseOut)
	case "insert":
		err = rewrite.Insert(sourcePath, outPath, recursive, minVariadic, verboseOut)
	default:
		fmt.Fprintf(os.Stderr, "error: unknown command %q\n", command)
		printUsage()
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
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
  -minvariadic    Use specialized WrapWithNFuncParams functions instead of
                  preserving existing variadic WrapWithFuncParams calls
  -verbose        Print progress information
  -help           Show help message

Examples:
  go-errs-wrap remove ./pkg/...
  go-errs-wrap replace ./pkg/mypackage/file.go
  go-errs-wrap insert ./pkg/mypackage/file.go
  go-errs-wrap replace -out ./output ./pkg/mypackage`)
}
