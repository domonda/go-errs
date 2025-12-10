/*
go-errs-wrap is a code transformation tool for managing error wrapping in Go.

It scans Go source files for defer errs.Wrap statements or //#wrap-result-err
marker comments and can either remove them or replace them with properly
generated defer errs.WrapWithFuncParams statements.

# Usage

	go-errs-wrap <command> [options] <path>

# Commands

	remove   Remove all defer errs.Wrap or //#wrap-result-err lines
	replace  Replace defer errs.Wrap or //#wrap-result-err with generated code

# Options

	-out <path>   Output to different location instead of modifying source
	-verbose      Print progress information
	-help         Show help message

# Examples

Remove all wrap statements from a directory:

	go-errs-wrap remove ./pkg/...

Replace wrap statements in a single file:

	go-errs-wrap replace ./pkg/mypackage/file.go

Replace and output to a different location:

	go-errs-wrap replace -out ./output ./pkg/mypackage
*/
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/domonda/go-errs/cmd/go-errs-wrap/rewrite"
)

var (
	outPath   string
	verbose   bool
	printHelp bool
)

func main() {
	flag.StringVar(&outPath, "out", "", "output to different location instead of modifying source")
	flag.BoolVar(&verbose, "verbose", false, "print progress information")
	flag.BoolVar(&printHelp, "help", false, "show help message")
	flag.Parse()

	if printHelp {
		printUsage()
		os.Exit(0)
	}

	args := flag.Args()
	if len(args) < 2 {
		fmt.Fprintln(os.Stderr, "error: missing command and/or path argument")
		printUsage()
		os.Exit(1)
	}

	command := args[0]
	sourcePath := args[1]

	var verboseOut io.Writer
	if verbose {
		verboseOut = os.Stdout
	}

	var err error
	switch command {
	case "remove":
		err = rewrite.Remove(sourcePath, outPath, verboseOut)
	case "replace":
		err = rewrite.Replace(sourcePath, outPath, verboseOut)
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

Arguments:
  path     Source directory or file path
           - If directory: recursively process all *.go files (excluding *_test.go)
           - If file: process only that file

Options:
  -out <path>   Output to different location instead of modifying source
                - If source is directory: create copy of directory structure
                - If source is file: write to specified file path
                - Non-Go files are copied unchanged
  -verbose      Print progress information
  -help         Show help message

Examples:
  go-errs-wrap remove ./pkg/...
  go-errs-wrap replace ./pkg/mypackage/file.go
  go-errs-wrap replace -out ./output ./pkg/mypackage`)
}
