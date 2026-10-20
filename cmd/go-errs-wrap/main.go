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

# Arguments

The path is a Go file, a directory for its package, or a directory followed
by /... for all packages below it, selected like the go command does.
Options must come before the path. See go-errs-wrap -help for details.

# Options

	-out <path>    Output to different location instead of modifying source.
	               A directory is copied with the rewritten files. Ignored with -validate.
	-validate      Dry run mode: check for issues without modifying files. Reports issues
	               to stderr and exits with error code 1 if any are found. Useful for CI
	               validation to ensure code quality standards are met.
	               - remove: checks if any defer errs.Wrap statements exist
	               - replace: checks if any defer errs.Wrap statements need updating
	               - insert: checks if any functions are missing defer errs.Wrap
	-minvariadic   Use specialized WrapWithNFuncParams functions instead of
	               preserving existing variadic WrapWithFuncParams calls
	-verbose       Print progress information
	-help          Show help message

# Exit Codes

	0   Success (no issues found in validate mode, or operation completed successfully)
	1   Error occurred (validation failed, file not found, invalid arguments, etc.)
	2   Invalid option (unknown flag or flag without value)

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
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
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
	if err := checkCommand(command); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		printUsage()
		os.Exit(1)
	}

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
	if err := checkOutFlag(fs); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	sourcePath, recursive, err := parsePathArgs(fs.Args())
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		printUsage()
		os.Exit(1)
	}

	var verboseOut io.Writer
	if verbose {
		verboseOut = os.Stdout
	}

	switch command {
	case "remove":
		err = rewrite.Remove(sourcePath, outPath, recursive, validate, verboseOut)
	case "replace":
		err = rewrite.Replace(sourcePath, outPath, recursive, minVariadic, validate, verboseOut)
	case "insert":
		err = rewrite.Insert(sourcePath, outPath, recursive, minVariadic, validate, verboseOut)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", errs.UnwrapCallStack(err))
		os.Exit(1)
	}
}

// checkCommand returns an error if command is not one of the commands.
// It is checked before the options are parsed, so that an option passed
// before the command, like -validate, is reported as unknown command
// instead of taking the command for the path.
func checkCommand(command string) error {
	switch command {
	case "remove", "replace", "insert":
		return nil
	}
	if strings.HasPrefix(command, "-") {
		return fmt.Errorf("unknown command %q, the command must come before the options", command)
	}
	return fmt.Errorf("unknown command %q", command)
}

// checkOutFlag returns an error if the -out flag was passed with an empty
// path, like an unset variable in a script, which would otherwise
// silently modify the source files in place.
func checkOutFlag(fs *flag.FlagSet) (err error) {
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "out" && f.Value.String() == "" {
			err = errors.New("-out needs a non-empty path")
		}
	})
	return err
}

// parsePathArgs returns the source path from the arguments left after
// parsing the flags, without a "..." pattern (e.g. "./..." or "./pkg/..."),
// and whether such a pattern requested to process the path recursively.
// The flag package stops parsing at the path, so an argument after the path
// is an error instead of a flag that is ignored, like -validate in CI.
func parsePathArgs(args []string) (sourcePath string, recursive bool, err error) {
	switch {
	case len(args) == 0:
		return "", false, errors.New("missing path argument")
	case len(args) > 1:
		return "", false, fmt.Errorf("unexpected argument %q after path %q, options must come before the path", args[1], args[0])
	}

	sourcePath = args[0]
	if sourcePath == "..." {
		return ".", true, nil
	}
	// The go command also accepts the separator of the OS, like .\pkg\... on Windows
	for _, suffix := range []string{"/...", string(filepath.Separator) + "..."} {
		if before, ok := strings.CutSuffix(sourcePath, suffix); ok {
			// Keep the separator of a root like / or C:\
			if before == filepath.VolumeName(before) {
				before += suffix[:1]
			}
			return before, true, nil
		}
	}
	return sourcePath, false, nil
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
           - If file: process only that file (*_test.go files are never processed)
           - If directory: process the Go package in that directory
           - If directory/...: process all packages below the directory
           Files are selected like the go command does: main packages are
           included, a directory skips files that need custom build tags and
           files starting with . or _, and /... also skips testdata, vendor,
           node_modules, directories starting with . or _, directories ignored
           by go.mod and nested modules. Unlike the go command, /... on a
           directory that is not within a module processes all modules below it.

Options:
  -out <path>     Output to different location instead of modifying source
                  - If source is directory: create a copy of the directory,
                    or of the directory tree with /..., with the rewritten files.
                    All other files are copied unchanged, except version control
                    metadata, symbolic links to directories and special files.
                    The output path must not be the source directory, be within
                    it or contain it, and its symbolic links to directories
                    must not lead outside of it.
                  - If source is file: write to the output path if it ends
                    with .go, else into that directory, created if missing.
                    The output file must not be the source file.
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
  2   Invalid option (unknown flag or flag without value)

Examples:
  go-errs-wrap remove ./pkg/...
  go-errs-wrap replace ./pkg/mypackage/file.go
  go-errs-wrap insert ./pkg/mypackage/file.go
  go-errs-wrap replace -out ./output ./pkg/mypackage
  go-errs-wrap remove -validate ./pkg/...
  go-errs-wrap replace -validate ./pkg/...
  go-errs-wrap insert -validate ./pkg/...`)
}
