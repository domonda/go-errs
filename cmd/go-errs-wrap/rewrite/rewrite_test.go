package rewrite

import (
	"go/ast"
	"go/parser"
	"go/token"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/ungerik/go-astvisit"
)

func TestIsDeferErrsWrap(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		aliases map[string]bool
		want    bool
	}{
		{
			name: "defer errs.WrapWithFuncParams",
			code: `package test
func f() { defer errs.WrapWithFuncParams(&err, x) }`,
			want: true,
		},
		{
			name: "defer errs.WrapWith0FuncParams",
			code: `package test
func f() { defer errs.WrapWith0FuncParams(&err) }`,
			want: true,
		},
		{
			name: "defer errs.WrapWith1FuncParam",
			code: `package test
func f() { defer errs.WrapWith1FuncParam(&err, x) }`,
			want: true,
		},
		{
			name: "defer errs.Wrap (generic)",
			code: `package test
func f() { defer errs.Wrap(&err) }`,
			want: true,
		},
		{
			name: "defer errs.WrapWithCallStack",
			code: `package test
func f() { defer errs.WrapWithCallStack(&err) }`,
			want: true,
		},
		{
			name: "defer other function",
			code: `package test
func f() { defer fmt.Println("done") }`,
			want: false,
		},
		{
			name: "defer close",
			code: `package test
func f() { defer file.Close() }`,
			want: false,
		},
		{
			name: "defer errors.Wrap (different package)",
			code: `package test
func f() { defer errors.Wrap(err) }`,
			want: false,
		},
		{
			name: "defer with alias",
			code: `package test
func f() { defer e.WrapWithFuncParams(&err, x) }`,
			aliases: map[string]bool{"e": true},
			want:    true,
		},
		{
			name: "defer with alias not matching",
			code: `package test
func f() { defer e.WrapWithFuncParams(&err, x) }`,
			aliases: map[string]bool{"errs": true},
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "test.go", tt.code, 0)
			require.NoError(t, err)

			var deferStmt *ast.DeferStmt
			ast.Inspect(file, func(n ast.Node) bool {
				if ds, ok := n.(*ast.DeferStmt); ok {
					deferStmt = ds
					return false
				}
				return true
			})
			require.NotNil(t, deferStmt, "no defer statement found")

			aliases := tt.aliases
			if aliases == nil {
				aliases = map[string]bool{"errs": true}
			}
			result := isDeferErrsWrap(deferStmt, aliases)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestIsVariadicWrapWithFuncParams(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		aliases map[string]bool
		want    bool
	}{
		{
			name: "variadic WrapWithFuncParams",
			code: `package test
func f() { defer errs.WrapWithFuncParams(&err, ctx, id) }`,
			want: true,
		},
		{
			name: "WrapWith0FuncParams",
			code: `package test
func f() { defer errs.WrapWith0FuncParams(&err) }`,
			want: false,
		},
		{
			name: "WrapWith1FuncParam",
			code: `package test
func f() { defer errs.WrapWith1FuncParam(&err, x) }`,
			want: false,
		},
		{
			name: "WrapWith2FuncParams",
			code: `package test
func f() { defer errs.WrapWith2FuncParams(&err, x, y) }`,
			want: false,
		},
		{
			name: "WrapWith10FuncParams",
			code: `package test
func f() { defer errs.WrapWith10FuncParams(&err, a, b, c, d, e, f, g, h, i, j) }`,
			want: false,
		},
		{
			name: "different package",
			code: `package test
func f() { defer errors.WrapWithFuncParams(&err, x) }`,
			want: false,
		},
		{
			name: "variadic with alias",
			code: `package test
func f() { defer e.WrapWithFuncParams(&err, x) }`,
			aliases: map[string]bool{"e": true},
			want:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "test.go", tt.code, 0)
			require.NoError(t, err)

			var deferStmt *ast.DeferStmt
			ast.Inspect(file, func(n ast.Node) bool {
				if ds, ok := n.(*ast.DeferStmt); ok {
					deferStmt = ds
					return false
				}
				return true
			})
			require.NotNil(t, deferStmt, "no defer statement found")

			aliases := tt.aliases
			if aliases == nil {
				aliases = map[string]bool{"errs": true}
			}
			result := isVariadicWrapWithFuncParams(deferStmt, aliases)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestReplacePreservesVariadic(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "go-errs-wrap-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create test input file with variadic WrapWithFuncParams
	inputCode := `package test

import "github.com/domonda/go-errs"

func ProcessData(ctx context.Context, id string) (err error) {
	defer errs.WrapWithFuncParams(&err, ctx) // variadic, should stay variadic

	return nil
}

func ProcessWithSpecialized(ctx context.Context) (err error) {
	defer errs.WrapWith1FuncParam(&err, ctx) // specialized, should become specialized

	return nil
}
`

	inputFile := filepath.Join(tmpDir, "input.go")
	err = os.WriteFile(inputFile, []byte(inputCode), 0644)
	require.NoError(t, err)

	// Create output directory
	outDir := filepath.Join(tmpDir, "output")
	err = os.MkdirAll(outDir, 0755)
	require.NoError(t, err)

	// Run replace with minVariadic=false to preserve variadic calls
	err = Replace(inputFile, outDir, false, false, false, nil)
	require.NoError(t, err)

	// Read output
	outputFile := filepath.Join(outDir, "input.go")
	output, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	outputStr := string(output)

	// Variadic should stay variadic with updated params
	assert.Contains(t, outputStr, "defer errs.WrapWithFuncParams(&err, ctx, id)")

	// Specialized should stay specialized with updated params
	assert.Contains(t, outputStr, "defer errs.WrapWith1FuncParam(&err, ctx)")
}

func TestReplaceMinVariadic(t *testing.T) {
	// Test that minVariadic=true converts variadic to specialized
	tmpDir, err := os.MkdirTemp("", "go-errs-wrap-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	inputCode := `package test

import "github.com/domonda/go-errs"

func VariadicFunc(ctx any, id string) (err error) {
	defer errs.WrapWithFuncParams(&err, ctx, id)
	return nil
}
`

	inputFile := filepath.Join(tmpDir, "input.go")
	err = os.WriteFile(inputFile, []byte(inputCode), 0644)
	require.NoError(t, err)

	outDir := filepath.Join(tmpDir, "output")
	err = os.MkdirAll(outDir, 0755)
	require.NoError(t, err)

	// Run replace with minVariadic=true to convert variadic to specialized
	err = Replace(inputFile, outDir, false, true, false, nil)
	require.NoError(t, err)

	outputFile := filepath.Join(outDir, "input.go")
	output, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	outputStr := string(output)

	// Variadic should be converted to specialized WrapWith2FuncParams
	assert.Contains(t, outputStr, "defer errs.WrapWith2FuncParams(&err, ctx, id)")
	assert.NotContains(t, outputStr, "defer errs.WrapWithFuncParams(&err")
}

func TestRemove(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "go-errs-wrap-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create test input file
	inputCode := `package test

import "github.com/domonda/go-errs"

func ProcessData(ctx context.Context, id string) (err error) {
	defer errs.WrapWithFuncParams(&err, ctx, id)

	return nil
}

func LoadFile(path string) (data []byte, err error) {
	//#wrap-result-err

	return nil, nil
}

func NoWrap(x int) error {
	return nil
}
`

	inputFile := filepath.Join(tmpDir, "input.go")
	err = os.WriteFile(inputFile, []byte(inputCode), 0644)
	require.NoError(t, err)

	// Create output directory
	outDir := filepath.Join(tmpDir, "output")
	err = os.MkdirAll(outDir, 0755)
	require.NoError(t, err)

	// Run remove
	err = Remove(inputFile, outDir, false, false, nil)
	require.NoError(t, err)

	// Read output
	outputFile := filepath.Join(outDir, "input.go")
	output, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	outputStr := string(output)

	// Verify defer statement was removed
	assert.NotContains(t, outputStr, "defer errs.WrapWithFuncParams")

	// Verify marker comment was removed
	assert.NotContains(t, outputStr, "//#wrap-result-err")

	// Verify other code remains
	assert.Contains(t, outputStr, "func ProcessData")
	assert.Contains(t, outputStr, "func LoadFile")
	assert.Contains(t, outputStr, "func NoWrap")
}

func TestReplace(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "go-errs-wrap-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create test input file
	inputCode := `package test

func ProcessData(ctx context.Context, id string) (err error) {
	defer errs.WrapWithFuncParams(&err, ctx) // outdated params

	return nil
}

func LoadFile(path string) (data []byte, err error) {
	//#wrap-result-err

	return nil, nil
}

func NoParams() (err error) {
	//#wrap-result-err
	return nil
}
`

	inputFile := filepath.Join(tmpDir, "input.go")
	err = os.WriteFile(inputFile, []byte(inputCode), 0644)
	require.NoError(t, err)

	// Create output directory
	outDir := filepath.Join(tmpDir, "output")
	err = os.MkdirAll(outDir, 0755)
	require.NoError(t, err)

	// Run replace with minVariadic=false to preserve variadic calls
	err = Replace(inputFile, outDir, false, false, false, nil)
	require.NoError(t, err)

	// Read output
	outputFile := filepath.Join(outDir, "input.go")
	output, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	outputStr := string(output)

	// Verify updated defer statement - stays variadic because input was variadic
	assert.Contains(t, outputStr, "defer errs.WrapWithFuncParams(&err, ctx, id)")

	// Verify replaced marker comment
	assert.NotContains(t, outputStr, "//#wrap-result-err")
	assert.Contains(t, outputStr, "defer errs.WrapWith1FuncParam(&err, path)")

	// Verify 0-param function
	assert.Contains(t, outputStr, "defer errs.WrapWith0FuncParams(&err)")
}

func TestReplaceNoNamedError(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "go-errs-wrap-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create test input file with function that has no named error result
	inputCode := `package test

func NoNamedErr(x int) error {
	//#wrap-result-err
	return nil
}
`

	inputFile := filepath.Join(tmpDir, "input.go")
	err = os.WriteFile(inputFile, []byte(inputCode), 0644)
	require.NoError(t, err)

	// Create output directory
	outDir := filepath.Join(tmpDir, "output")
	err = os.MkdirAll(outDir, 0755)
	require.NoError(t, err)

	// Run replace (should not error, but should skip the function)
	err = Replace(inputFile, outDir, false, true, false, nil)
	require.NoError(t, err)

	// Read output - should be unchanged since the function was skipped
	outputFile := filepath.Join(outDir, "input.go")
	_, err = os.ReadFile(outputFile)
	// File may not exist or may be unchanged since there were no valid replacements
	// This is expected behavior
}

func TestProcessFileWithAnonymousFunction(t *testing.T) {
	code := `package test

func Outer(a int) (err error) {
	inner := func(b string) (innerErr error) {
		//#wrap-result-err
		return nil
	}
	_ = inner
	return nil
}
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", code, parser.ParseComments)
	require.NoError(t, err)

	replacements, imports, err := processFile(fset, file, true, nil, modeReplace)
	require.NoError(t, err)

	// Should have one replacement for the anonymous function
	assert.Len(t, replacements, 1)
	assert.Contains(t, imports, astvisit.Import{Path: "github.com/domonda/go-errs"})
}

func TestCopyFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "go-errs-wrap-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create source file
	srcContent := []byte("test content")
	srcPath := filepath.Join(tmpDir, "source.txt")
	err = os.WriteFile(srcPath, srcContent, 0644)
	require.NoError(t, err)

	// Copy file
	dstPath := filepath.Join(tmpDir, "dest.txt")
	root, err := os.OpenRoot(tmpDir)
	require.NoError(t, err)
	defer root.Close()
	err = copyFile(srcPath, root, "dest.txt", 0644)
	require.NoError(t, err)

	// Verify content
	dstContent, err := os.ReadFile(dstPath)
	require.NoError(t, err)
	assert.Equal(t, srcContent, dstContent)
}

func TestMarkerCommentVariations(t *testing.T) {
	tests := []struct {
		name    string
		comment string
		isMatch bool
	}{
		{
			name:    "exact match",
			comment: "//#wrap-result-err",
			isMatch: true,
		},
		{
			name:    "with leading space after //",
			comment: "// #wrap-result-err",
			isMatch: true, // trimmed becomes "#wrap-result-err" which matches
		},
		{
			name:    "with trailing space",
			comment: "//#wrap-result-err ",
			isMatch: true,
		},
		{
			name:    "different comment",
			comment: "// some other comment",
			isMatch: false,
		},
		{
			name:    "partial match",
			comment: "//#wrap-result-err and more",
			isMatch: false,
		},
		{
			name:    "without hash",
			comment: "//wrap-result-err",
			isMatch: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trimmed := strings.TrimSpace(strings.TrimPrefix(tt.comment, "//"))
			isMatch := trimmed == "#wrap-result-err"
			assert.Equal(t, tt.isMatch, isMatch)
		})
	}
}

func TestInsert(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "go-errs-wrap-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create test input file without any defer errs.Wrap statements
	inputCode := `package test

func ProcessData(ctx context.Context, id string) (err error) {
	return nil
}

func LoadFile(path string) (data []byte, err error) {
	return nil, nil
}

func NoNamedErr(x int) error {
	return nil
}

func NoError(x int) int {
	return x
}
`

	inputFile := filepath.Join(tmpDir, "input.go")
	err = os.WriteFile(inputFile, []byte(inputCode), 0644)
	require.NoError(t, err)

	// Create output directory
	outDir := filepath.Join(tmpDir, "output")
	err = os.MkdirAll(outDir, 0755)
	require.NoError(t, err)

	// Run insert
	err = Insert(inputFile, outDir, false, true, false, nil)
	require.NoError(t, err)

	// Read output
	outputFile := filepath.Join(outDir, "input.go")
	output, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	outputStr := string(output)

	// Verify defer statements were inserted for functions with named error results
	assert.Contains(t, outputStr, "defer errs.WrapWith2FuncParams(&err, ctx, id)")
	assert.Contains(t, outputStr, "defer errs.WrapWith1FuncParam(&err, path)")

	// Verify no defer was inserted for functions without named error result
	// or without error return type at all
	assert.NotContains(t, outputStr, "WrapWith1FuncParam(&err, x)") // NoNamedErr has unnamed error
}

func TestInsertSkipsExisting(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "go-errs-wrap-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create test input file with one function that already has defer errs.Wrap
	inputCode := `package test

import "github.com/domonda/go-errs"

func AlreadyWrapped(ctx context.Context) (err error) {
	defer errs.WrapWith1FuncParam(&err, ctx)

	return nil
}

func NeedsWrap(id string) (err error) {
	return nil
}
`

	inputFile := filepath.Join(tmpDir, "input.go")
	err = os.WriteFile(inputFile, []byte(inputCode), 0644)
	require.NoError(t, err)

	// Create output directory
	outDir := filepath.Join(tmpDir, "output")
	err = os.MkdirAll(outDir, 0755)
	require.NoError(t, err)

	// Run insert
	err = Insert(inputFile, outDir, false, true, false, nil)
	require.NoError(t, err)

	// Read output
	outputFile := filepath.Join(outDir, "input.go")
	output, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	outputStr := string(output)

	// AlreadyWrapped should still have exactly one defer (not duplicated)
	assert.Equal(t, 1, strings.Count(outputStr, "WrapWith1FuncParam(&err, ctx)"))

	// NeedsWrap should have the new defer inserted
	assert.Contains(t, outputStr, "WrapWith1FuncParam(&err, id)")
}

func TestInsertEmptyFunctionBody(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "go-errs-wrap-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create test input file with empty function body
	inputCode := `package test

func EmptyBody() (err error) {
}
`

	inputFile := filepath.Join(tmpDir, "input.go")
	err = os.WriteFile(inputFile, []byte(inputCode), 0644)
	require.NoError(t, err)

	// Create output directory
	outDir := filepath.Join(tmpDir, "output")
	err = os.MkdirAll(outDir, 0755)
	require.NoError(t, err)

	// Run insert
	err = Insert(inputFile, outDir, false, true, false, nil)
	require.NoError(t, err)

	// Read output
	outputFile := filepath.Join(outDir, "input.go")
	output, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	outputStr := string(output)

	// Verify defer statement was inserted even in empty function body
	assert.Contains(t, outputStr, "defer errs.WrapWith0FuncParams(&err)")
}

func TestInsertWithEmptyLineAfter(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "go-errs-wrap-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create test input file
	inputCode := `package test

func ProcessData(id string) (err error) {
	x := 1
	return nil
}
`

	inputFile := filepath.Join(tmpDir, "input.go")
	err = os.WriteFile(inputFile, []byte(inputCode), 0644)
	require.NoError(t, err)

	// Create output directory
	outDir := filepath.Join(tmpDir, "output")
	err = os.MkdirAll(outDir, 0755)
	require.NoError(t, err)

	// Run insert
	err = Insert(inputFile, outDir, false, true, false, nil)
	require.NoError(t, err)

	// Read output
	outputFile := filepath.Join(outDir, "input.go")
	output, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	outputStr := string(output)

	// Verify there's an empty line after the defer statement
	// The pattern should be: defer...(&err, id)\n\n\t(next statement)
	assert.Contains(t, outputStr, "defer errs.WrapWith1FuncParam(&err, id)\n\n")
}

func TestExtractErrsAliases(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected map[string]bool
	}{
		{
			name:     "no import",
			code:     `package test`,
			expected: map[string]bool{"errs": true}, // default fallback
		},
		{
			name: "default import",
			code: `package test
import "github.com/domonda/go-errs"`,
			expected: map[string]bool{"errs": true},
		},
		{
			name: "aliased import",
			code: `package test
import e "github.com/domonda/go-errs"`,
			expected: map[string]bool{"e": true},
		},
		{
			name: "named import",
			code: `package test
import goerrs "github.com/domonda/go-errs"`,
			expected: map[string]bool{"goerrs": true},
		},
		{
			name: "multiple imports with alias",
			code: `package test
import (
	"fmt"
	myerrs "github.com/domonda/go-errs"
	"os"
)`,
			expected: map[string]bool{"myerrs": true},
		},
		{
			name: "other imports only",
			code: `package test
import (
	"fmt"
	"os"
)`,
			expected: map[string]bool{"errs": true}, // default fallback
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "test.go", tt.code, parser.ImportsOnly)
			require.NoError(t, err)

			result := extractErrsAliases(file)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractKeepSecretParams(t *testing.T) {
	tests := []struct {
		name     string
		code     string
		expected map[string]bool
	}{
		{
			name: "no KeepSecret",
			code: `package test
func f() { defer errs.WrapWithFuncParams(&err, ctx, id) }`,
			expected: map[string]bool{},
		},
		{
			name: "one KeepSecret",
			code: `package test
func f() { defer errs.WrapWithFuncParams(&err, ctx, errs.KeepSecret(password)) }`,
			expected: map[string]bool{"password": true},
		},
		{
			name: "multiple KeepSecret",
			code: `package test
func f() { defer errs.WrapWithFuncParams(&err, errs.KeepSecret(apiKey), ctx, errs.KeepSecret(token)) }`,
			expected: map[string]bool{"apiKey": true, "token": true},
		},
		{
			name: "KeepSecret with specialized function",
			code: `package test
func f() { defer errs.WrapWith2FuncParams(&err, ctx, errs.KeepSecret(secret)) }`,
			expected: map[string]bool{"secret": true},
		},
		{
			name: "all params KeepSecret",
			code: `package test
func f() { defer errs.WrapWith2FuncParams(&err, errs.KeepSecret(key), errs.KeepSecret(pwd)) }`,
			expected: map[string]bool{"key": true, "pwd": true},
		},
		{
			name: "KeepSecret with aliased import",
			code: `package test
import e "github.com/domonda/go-errs"
func f() { defer e.WrapWithFuncParams(&err, ctx, e.KeepSecret(password)) }`,
			expected: map[string]bool{"password": true},
		},
		{
			name: "KeepSecret with named import",
			code: `package test
import goerrs "github.com/domonda/go-errs"
func f() { defer goerrs.WrapWith2FuncParams(&err, goerrs.KeepSecret(key), ctx) }`,
			expected: map[string]bool{"key": true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "test.go", tt.code, 0)
			require.NoError(t, err)

			var deferStmt *ast.DeferStmt
			ast.Inspect(file, func(n ast.Node) bool {
				if ds, ok := n.(*ast.DeferStmt); ok {
					deferStmt = ds
					return false
				}
				return true
			})
			require.NotNil(t, deferStmt, "no defer statement found")

			errsAliases := extractErrsAliases(file)
			result := extractKeepSecretParams(deferStmt, errsAliases)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestReplacePreservesKeepSecret(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "go-errs-wrap-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create test input file with KeepSecret-wrapped parameters
	inputCode := `package test

import "github.com/domonda/go-errs"

func ProcessWithSecret(ctx context.Context, password string) (err error) {
	defer errs.WrapWithFuncParams(&err, ctx, errs.KeepSecret(password))

	return nil
}

func MultipleSecrets(apiKey, token, userID string) (err error) {
	defer errs.WrapWith3FuncParams(&err, errs.KeepSecret(apiKey), errs.KeepSecret(token), userID)

	return nil
}
`

	inputFile := filepath.Join(tmpDir, "input.go")
	err = os.WriteFile(inputFile, []byte(inputCode), 0644)
	require.NoError(t, err)

	// Create output directory
	outDir := filepath.Join(tmpDir, "output")
	err = os.MkdirAll(outDir, 0755)
	require.NoError(t, err)

	// Run replace (preserving variadic)
	err = Replace(inputFile, outDir, false, false, false, nil)
	require.NoError(t, err)

	// Read output
	outputFile := filepath.Join(outDir, "input.go")
	output, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	outputStr := string(output)

	// Verify KeepSecret is preserved for password
	assert.Contains(t, outputStr, "errs.KeepSecret(password)")
	assert.NotContains(t, outputStr, "&err, ctx, password)") // Should not have unwrapped password

	// Verify multiple KeepSecret are preserved
	assert.Contains(t, outputStr, "errs.KeepSecret(apiKey)")
	assert.Contains(t, outputStr, "errs.KeepSecret(token)")
	assert.NotContains(t, outputStr, "errs.KeepSecret(userID)") // userID was not wrapped

	// Verify userID is present unwrapped (as expected)
	assert.Contains(t, outputStr, "userID)")
}

func TestReplacePreservesKeepSecretWithMinVariadic(t *testing.T) {
	// Test that KeepSecret is preserved even with minVariadic=true
	tmpDir, err := os.MkdirTemp("", "go-errs-wrap-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	inputCode := `package test

import "github.com/domonda/go-errs"

func ProcessWithSecret(ctx any, password string) (err error) {
	defer errs.WrapWithFuncParams(&err, ctx, errs.KeepSecret(password))
	return nil
}
`

	inputFile := filepath.Join(tmpDir, "input.go")
	err = os.WriteFile(inputFile, []byte(inputCode), 0644)
	require.NoError(t, err)

	outDir := filepath.Join(tmpDir, "output")
	err = os.MkdirAll(outDir, 0755)
	require.NoError(t, err)

	// Run replace with minVariadic=true
	err = Replace(inputFile, outDir, false, true, false, nil)
	require.NoError(t, err)

	outputFile := filepath.Join(outDir, "input.go")
	output, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	outputStr := string(output)

	// Should convert to specialized function but preserve KeepSecret
	assert.Contains(t, outputStr, "WrapWith2FuncParams(&err, ctx, errs.KeepSecret(password))")
}

func TestGenerateWrapStatementWithKeepSecret(t *testing.T) {
	tests := []struct {
		name             string
		funcInfo         *funcInfo
		expectedVariadic string
		expectedSpecial  string
	}{
		{
			name: "one secret param",
			funcInfo: &funcInfo{
				funcName:        "test",
				paramNames:      []string{"ctx", "password"},
				keepSecretNames: map[string]bool{"password": true},
				errorResultName: "err",
			},
			expectedVariadic: "defer errs.WrapWithFuncParams(&err, ctx, errs.KeepSecret(password))",
			expectedSpecial:  "defer errs.WrapWith2FuncParams(&err, ctx, errs.KeepSecret(password))",
		},
		{
			name: "multiple secret params",
			funcInfo: &funcInfo{
				funcName:        "test",
				paramNames:      []string{"apiKey", "ctx", "token"},
				keepSecretNames: map[string]bool{"apiKey": true, "token": true},
				errorResultName: "err",
			},
			expectedVariadic: "defer errs.WrapWithFuncParams(&err, errs.KeepSecret(apiKey), ctx, errs.KeepSecret(token))",
			expectedSpecial:  "defer errs.WrapWith3FuncParams(&err, errs.KeepSecret(apiKey), ctx, errs.KeepSecret(token))",
		},
		{
			name: "all params secret",
			funcInfo: &funcInfo{
				funcName:        "test",
				paramNames:      []string{"key", "secret"},
				keepSecretNames: map[string]bool{"key": true, "secret": true},
				errorResultName: "err",
			},
			expectedVariadic: "defer errs.WrapWithFuncParams(&err, errs.KeepSecret(key), errs.KeepSecret(secret))",
			expectedSpecial:  "defer errs.WrapWith2FuncParams(&err, errs.KeepSecret(key), errs.KeepSecret(secret))",
		},
		{
			name: "single secret param",
			funcInfo: &funcInfo{
				funcName:        "test",
				paramNames:      []string{"password"},
				keepSecretNames: map[string]bool{"password": true},
				errorResultName: "err",
			},
			expectedVariadic: "defer errs.WrapWithFuncParams(&err, errs.KeepSecret(password))",
			expectedSpecial:  "defer errs.WrapWith1FuncParam(&err, errs.KeepSecret(password))",
		},
		{
			name: "no secret params",
			funcInfo: &funcInfo{
				funcName:        "test",
				paramNames:      []string{"ctx", "id"},
				keepSecretNames: nil,
				errorResultName: "err",
			},
			expectedVariadic: "defer errs.WrapWithFuncParams(&err, ctx, id)",
			expectedSpecial:  "defer errs.WrapWith2FuncParams(&err, ctx, id)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			variadic := generateVariadicWrapStatement(tt.funcInfo)
			assert.Equal(t, tt.expectedVariadic, variadic)

			special := generateWrapStatement(tt.funcInfo)
			assert.Equal(t, tt.expectedSpecial, special)
		})
	}
}

func TestReplaceValidateMode(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "go-errs-wrap-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create test input file with outdated wrap statements
	inputCode := `package test

func ProcessData(ctx context.Context, id string) (err error) {
	defer errs.WrapWithFuncParams(&err, ctx) // outdated params

	return nil
}

func LoadFile(path string) (data []byte, err error) {
	//#wrap-result-err

	return nil, nil
}
`

	inputFile := filepath.Join(tmpDir, "input.go")
	err = os.WriteFile(inputFile, []byte(inputCode), 0644)
	require.NoError(t, err)

	// Run replace with validate=true
	err = Replace(inputFile, "", false, false, true, nil)

	// Should return an error indicating missing replacements
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing error wrapper")

	// Verify file was not modified
	content, readErr := os.ReadFile(inputFile)
	require.NoError(t, readErr)
	assert.Equal(t, inputCode, string(content))
}

func TestReplaceValidateModeNoIssues(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "go-errs-wrap-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create test input file with correct wrap statements
	inputCode := `package test

import "github.com/domonda/go-errs"

func ProcessData(ctx context.Context, id string) (err error) {
	defer errs.WrapWith2FuncParams(&err, ctx, id)

	return nil
}

func NoError() {
	return
}
`

	inputFile := filepath.Join(tmpDir, "input.go")
	err = os.WriteFile(inputFile, []byte(inputCode), 0644)
	require.NoError(t, err)

	// Run replace with validate=true
	err = Replace(inputFile, "", false, false, true, nil)

	// Should succeed with no errors
	require.NoError(t, err)

	// Verify file was not modified
	content, readErr := os.ReadFile(inputFile)
	require.NoError(t, readErr)
	assert.Equal(t, inputCode, string(content))
}

func TestReplaceValidateModeZeroParamVariadic(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "go-errs-wrap-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	inputCode := `package test

import "github.com/domonda/go-errs"

func NoParams() (err error) {
	defer errs.WrapWithFuncParams(&err)

	return nil
}
`

	inputFile := filepath.Join(tmpDir, "input.go")
	err = os.WriteFile(inputFile, []byte(inputCode), 0644)
	require.NoError(t, err)

	err = Replace(inputFile, "", false, false, true, nil)
	require.NoError(t, err)

	content, readErr := os.ReadFile(inputFile)
	require.NoError(t, readErr)
	assert.Equal(t, inputCode, string(content))
}

func TestInsertValidateMode(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "go-errs-wrap-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create test input file without defer statements
	inputCode := `package test

func ProcessData(ctx context.Context, id string) (err error) {
	return nil
}

func LoadFile(path string) (data []byte, err error) {
	return nil, nil
}
`

	inputFile := filepath.Join(tmpDir, "input.go")
	err = os.WriteFile(inputFile, []byte(inputCode), 0644)
	require.NoError(t, err)

	// Run insert with validate=true
	err = Insert(inputFile, "", false, true, true, nil)

	// Should return an error indicating missing insertions
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing error wrapper")

	// Verify file was not modified
	content, readErr := os.ReadFile(inputFile)
	require.NoError(t, readErr)
	assert.Equal(t, inputCode, string(content))
}

func TestInsertValidateModeNoIssues(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "go-errs-wrap-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create test input file with all required defer statements
	inputCode := `package test

import "github.com/domonda/go-errs"

func ProcessData(ctx context.Context, id string) (err error) {
	defer errs.WrapWith2FuncParams(&err, ctx, id)

	return nil
}

func LoadFile(path string) (data []byte, err error) {
	defer errs.WrapWith1FuncParam(&err, path)

	return nil, nil
}

func NoError() int {
	return 42
}
`

	inputFile := filepath.Join(tmpDir, "input.go")
	err = os.WriteFile(inputFile, []byte(inputCode), 0644)
	require.NoError(t, err)

	// Run insert with validate=true
	err = Insert(inputFile, "", false, true, true, nil)

	// Should succeed with no errors
	require.NoError(t, err)

	// Verify file was not modified
	content, readErr := os.ReadFile(inputFile)
	require.NoError(t, readErr)
	assert.Equal(t, inputCode, string(content))
}

func TestValidateIgnoresOutPath(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "go-errs-wrap-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	inputCode := `package test
func F() (err error) { return nil }
`

	inputFile := filepath.Join(tmpDir, "input.go")
	err = os.WriteFile(inputFile, []byte(inputCode), 0644)
	require.NoError(t, err)

	outDir := filepath.Join(tmpDir, "output")
	err = os.MkdirAll(outDir, 0755)
	require.NoError(t, err)

	// Run remove with validate=true and outPath set (should ignore outPath and succeed - nothing to remove)
	err = Remove(inputFile, outDir, false, true, nil)
	require.NoError(t, err)

	// Run replace with validate=true and outPath set (should ignore outPath and succeed - nothing to replace)
	err = Replace(inputFile, outDir, false, false, true, nil)
	require.NoError(t, err)

	// Run insert with validate=true and outPath set (should ignore outPath but FAIL - missing wrapper)
	err = Insert(inputFile, outDir, false, false, true, nil)
	require.Error(t, err, "insert validation should fail when wrappers are missing")
	assert.Contains(t, err.Error(), "missing error wrapper")

	// Verify that outDir is still empty (nothing was written even though insert validation failed)
	entries, err := os.ReadDir(outDir)
	require.NoError(t, err)
	assert.Empty(t, entries, "output directory should be empty when using -validate")
}

func TestRemoveValidateMode(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "go-errs-wrap-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create test input file with defer errs.Wrap statements
	inputCode := `package test

import "github.com/domonda/go-errs"

func ProcessData(ctx context.Context, id string) (err error) {
	defer errs.WrapWith2FuncParams(&err, ctx, id)

	return nil
}

func LoadFile(path string) (data []byte, err error) {
	//#wrap-result-err

	return nil, nil
}
`

	inputFile := filepath.Join(tmpDir, "input.go")
	err = os.WriteFile(inputFile, []byte(inputCode), 0644)
	require.NoError(t, err)

	// Run remove with validate=true
	err = Remove(inputFile, "", false, true, nil)

	// Should return an error indicating defer statements exist
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing error wrapper")

	// Verify file was not modified
	content, readErr := os.ReadFile(inputFile)
	require.NoError(t, readErr)
	assert.Equal(t, inputCode, string(content))
}

func TestRemoveValidateModeNoIssues(t *testing.T) {
	// Create a temporary directory for test files
	tmpDir, err := os.MkdirTemp("", "go-errs-wrap-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create test input file without any defer errs.Wrap statements
	inputCode := `package test

func ProcessData(ctx context.Context, id string) (err error) {
	return nil
}

func NoError() {
	return
}
`

	inputFile := filepath.Join(tmpDir, "input.go")
	err = os.WriteFile(inputFile, []byte(inputCode), 0644)
	require.NoError(t, err)

	// Run remove with validate=true
	err = Remove(inputFile, "", false, true, nil)

	// Should succeed with no errors
	require.NoError(t, err)

	// Verify file was not modified
	content, readErr := os.ReadFile(inputFile)
	require.NoError(t, readErr)
	assert.Equal(t, inputCode, string(content))
}

// TestReplaceValidateModeUnsortedImports verifies that replace -validate
// does NOT report missing wrappers just because imports are not sorted.
// Import reordering is a formatting concern handled by goimports/gofmt, not
// a concern of the error-wrap validator. Regression test for a bug where
// any whole-file formatting difference (including pure import reordering)
// caused every already-correct defer errs.Wrap to be flagged as "missing".
func TestReplaceValidateModeUnsortedImports(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "go-errs-wrap-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// All wrap statements are already correct, but the imports within the
	// domonda group are not alphabetically sorted (domonda-service/...
	// appears after go-errs/golog instead of before them).
	inputCode := `package test

import (
	"context"

	"github.com/domonda/go-errs"
	"github.com/domonda/golog"
	"github.com/domonda/example-project/pkg/object"
)

var _ = golog.CallingFunctionName
var _ = object.ClassRealEstateObject

func ProcessData(ctx context.Context, id string) (err error) {
	defer errs.WrapWithFuncParams(&err, ctx, id)

	return nil
}
`

	inputFile := filepath.Join(tmpDir, "input.go")
	err = os.WriteFile(inputFile, []byte(inputCode), 0644)
	require.NoError(t, err)

	// Validate should succeed: the defer statement itself is already correct.
	err = Replace(inputFile, "", false, false, true, nil)
	require.NoError(t, err, "unsorted imports alone must not trigger a missing-wrapper error")

	// File must not be modified in validate mode.
	content, readErr := os.ReadFile(inputFile)
	require.NoError(t, readErr)
	assert.Equal(t, inputCode, string(content))
}

// TestReplaceNoOpDoesNotReorderImports verifies that running replace (without
// -validate) on a file whose wrap statements are all already correct does NOT
// modify the file — even if the file's imports are not in the order that
// goimports would produce. The tool's job is to fix wrap statements, not to
// reformat unrelated code; import sorting belongs to gofmt/goimports.
func TestReplaceNoOpDoesNotReorderImports(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "go-errs-wrap-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Imports within the domonda group are NOT alphabetically sorted.
	// All defer wrap statements are already correct.
	inputCode := `package test

import (
	"context"

	"github.com/domonda/go-errs"
	"github.com/domonda/golog"
	"github.com/domonda/example-project/pkg/object"
)

var _ = golog.CallingFunctionName
var _ = object.ClassRealEstateObject

func ProcessData(ctx context.Context, id string) (err error) {
	defer errs.WrapWithFuncParams(&err, ctx, id)

	return nil
}
`

	inputFile := filepath.Join(tmpDir, "input.go")
	err = os.WriteFile(inputFile, []byte(inputCode), 0644)
	require.NoError(t, err)

	// Run replace in normal (non-validate) mode.
	err = Replace(inputFile, "", false, false, false, nil)
	require.NoError(t, err)

	// File must be byte-for-byte unchanged: no replacements were needed,
	// so import order should not have been touched.
	content, readErr := os.ReadFile(inputFile)
	require.NoError(t, readErr)
	assert.Equal(t, inputCode, string(content), "no-op replace must not reorder imports")
}

// TestReplaceValidateModeReportsOnlyActuallyOutdatedWrappers verifies that
// when a file contains a mix of correct and outdated wrap statements and
// also has unsorted imports, validate reports ONLY the outdated one — not
// every defer in the file.
func TestReplaceValidateModeReportsOnlyActuallyOutdatedWrappers(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "go-errs-wrap-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Two functions: first has a correct wrapper, second is outdated
	// (missing the id parameter). Imports are also unsorted.
	inputCode := `package test

import (
	"context"

	"github.com/domonda/go-errs"
	"github.com/domonda/golog"
	"github.com/domonda/example-project/pkg/object"
)

var _ = golog.CallingFunctionName
var _ = object.ClassRealEstateObject

func Correct(ctx context.Context, id string) (err error) {
	defer errs.WrapWithFuncParams(&err, ctx, id)

	return nil
}

func Outdated(ctx context.Context, id string) (err error) {
	defer errs.WrapWithFuncParams(&err, ctx)

	return nil
}
`

	inputFile := filepath.Join(tmpDir, "input.go")
	err = os.WriteFile(inputFile, []byte(inputCode), 0644)
	require.NoError(t, err)

	err = Replace(inputFile, "", false, false, true, nil)
	require.Error(t, err)
	// Exactly one missing wrapper, not two.
	assert.Contains(t, err.Error(), "found 1 missing error wrapper")

	content, readErr := os.ReadFile(inputFile)
	require.NoError(t, readErr)
	assert.Equal(t, inputCode, string(content))
}

// TestRewriteAddsErrsImport verifies that every rewrite emitting an
// errs.WrapWith* statement imports go-errs into a file that lacks it
// and keeps an existing import without duplicating it, for all three ways
// of writing the result: in place, to an output file, and to an output directory.
// Each case without import is generated by a different branch of processFile.
func TestRewriteAddsErrsImport(t *testing.T) {
	replace := func(sourcePath, outPath string) error {
		return Replace(sourcePath, outPath, false, true, false, nil)
	}
	insert := func(sourcePath, outPath string) error {
		return Insert(sourcePath, outPath, false, true, false, nil)
	}
	tests := []struct {
		name     string
		rewrite  func(sourcePath, outPath string) error
		input    string
		wantWrap string
	}{
		{
			name:    "replace defer errs.Wrap",
			rewrite: replace,
			input: `package test

func Load(id string) (err error) {
	defer errs.WrapWith0FuncParams(&err)

	return nil
}
`,
			wantWrap: "defer errs.WrapWith1FuncParam(&err, id)",
		},
		{
			name:    "replace marker comment",
			rewrite: replace,
			input: `package test

func Load(id string) (err error) {
	//#wrap-result-err

	return nil
}
`,
			wantWrap: "defer errs.WrapWith1FuncParam(&err, id)",
		},
		{
			name:    "insert before first statement",
			rewrite: insert,
			input: `package test

func Load(id string) (err error) {
	return nil
}
`,
			wantWrap: "defer errs.WrapWith1FuncParam(&err, id)",
		},
		{
			name:    "insert into empty body",
			rewrite: insert,
			input: `package test

func Load() (err error) {
}
`,
			wantWrap: "defer errs.WrapWith0FuncParams(&err)",
		},
		{
			name:    "replace keeps existing import",
			rewrite: replace,
			input: `package test

import (
	"context"

	"github.com/domonda/go-errs"
)

func Load(ctx context.Context, id string) (err error) {
	defer errs.WrapWith1FuncParam(&err, ctx)

	return nil
}
`,
			wantWrap: "defer errs.WrapWith2FuncParams(&err, ctx, id)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Run("in place", func(t *testing.T) {
				file := filepath.Join(t.TempDir(), "load.go")
				err := os.WriteFile(file, []byte(tt.input), 0600)
				require.NoError(t, err)

				err = tt.rewrite(file, "")
				require.NoError(t, err)
				assertWrapAndErrsImport(t, file, tt.wantWrap)
			})
			t.Run("output file", func(t *testing.T) {
				file := filepath.Join(t.TempDir(), "load.go")
				err := os.WriteFile(file, []byte(tt.input), 0600)
				require.NoError(t, err)
				outFile := filepath.Join(t.TempDir(), "load.go")

				err = tt.rewrite(file, outFile)
				require.NoError(t, err)
				assertWrapAndErrsImport(t, outFile, tt.wantWrap)
			})
			t.Run("output directory", func(t *testing.T) {
				sourceDir := t.TempDir()
				err := os.WriteFile(filepath.Join(sourceDir, "load.go"), []byte(tt.input), 0600)
				require.NoError(t, err)
				outDir := t.TempDir()

				err = tt.rewrite(sourceDir, outDir)
				require.NoError(t, err)
				assertWrapAndErrsImport(t, filepath.Join(outDir, "load.go"), tt.wantWrap)
			})
		})
	}
}

// assertWrapAndErrsImport asserts that the Go file at path contains wantWrap
// and imports go-errs exactly once, without an import name.
func assertWrapAndErrsImport(t *testing.T, path, wantWrap string) {
	t.Helper()
	output, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(output), wantWrap)

	file, err := parser.ParseFile(token.NewFileSet(), path, output, parser.ImportsOnly)
	require.NoError(t, err)
	var errsImports []astvisit.Import
	for _, spec := range file.Imports {
		imp, err := astvisit.ImportOf(spec)
		require.NoError(t, err)
		if imp.Path == "github.com/domonda/go-errs" {
			errsImports = append(errsImports, imp)
		}
	}
	assert.Equal(t, []astvisit.Import{{Path: "github.com/domonda/go-errs"}}, errsImports)
}

// TestRewriteToOutputDirectory verifies rewriting a source directory
// recursively into an output directory: non-Go files are copied, Go files
// of subdirectories are written to the same relative path, Remove also drops
// the then unused go-errs import, verbose output names every copied
// and written file, and a failed write is returned as error.
func TestRewriteToOutputDirectory(t *testing.T) {
	sourceDir := t.TempDir()
	writeSourceFile := func(relPath, content string) {
		t.Helper()
		path := filepath.Join(sourceDir, relPath)
		err := os.MkdirAll(filepath.Dir(path), 0750)
		require.NoError(t, err)
		err = os.WriteFile(path, []byte(content), 0600)
		require.NoError(t, err)
	}
	writeSourceFile("README.md", "# Readme\n")
	writeSourceFile(filepath.Join("sub", "load.go"), `package sub

import "github.com/domonda/go-errs"

func Load(id string) (err error) {
	defer errs.WrapWithFuncParams(&err)

	return nil
}
`)

	t.Run("replace", func(t *testing.T) {
		outDir := t.TempDir()
		var verbose strings.Builder

		err := Replace(sourceDir, outDir, true, false, false, &verbose)
		require.NoError(t, err)

		readme, err := os.ReadFile(filepath.Join(outDir, "README.md"))
		require.NoError(t, err)
		assert.Equal(t, "# Readme\n", string(readme))
		outFile := filepath.Join(outDir, "sub", "load.go")
		assertWrapAndErrsImport(t, outFile, "defer errs.WrapWithFuncParams(&err, id)")

		assert.Contains(t, verbose.String(), "copying: "+filepath.Join(sourceDir, "README.md")+" -> "+filepath.Join(outDir, "README.md"))
		assert.Contains(t, verbose.String(), "wrote: "+outFile)
	})

	t.Run("remove", func(t *testing.T) {
		outDir := t.TempDir()

		err := Remove(sourceDir, outDir, true, false, nil)
		require.NoError(t, err)

		output, err := os.ReadFile(filepath.Join(outDir, "sub", "load.go"))
		require.NoError(t, err)
		assert.Equal(t, "package sub\n\nfunc Load(id string) (err error) {\n\n\treturn nil\n}\n", string(output))
	})

	t.Run("copy error", func(t *testing.T) {
		outDir := t.TempDir()
		// A directory in place of the copy of a file makes copying it fail
		err := os.MkdirAll(filepath.Join(outDir, "sub", "load.go"), 0750)
		require.NoError(t, err)

		err = Replace(sourceDir, outDir, true, false, false, nil)
		require.Error(t, err)
	})

	t.Run("read-only source", func(t *testing.T) {
		// Read-only sources, like the files of the module cache,
		// are rewritten into owner-writable files, so the output
		// can be written again
		readOnlySourceDir := t.TempDir()
		err := os.WriteFile(filepath.Join(readOnlySourceDir, "load.go"), []byte(`package test

func Load(id string) (err error) {
	defer errs.WrapWithFuncParams(&err)

	return nil
}
`), 0400)
		require.NoError(t, err)
		outDir := t.TempDir()

		for range 2 {
			err = Replace(readOnlySourceDir, outDir, false, false, false, nil)
			require.NoError(t, err)
		}
		assertWrapAndErrsImport(t, filepath.Join(outDir, "load.go"), "defer errs.WrapWithFuncParams(&err, id)")
	})
}

// TestRewriteSingleFileOutputPath verifies how the output path of a single
// source file is resolved: a missing path without .go suffix is created
// as directory for the output file, and a failed write is returned as error.
func TestRewriteSingleFileOutputPath(t *testing.T) {
	sourceFile := filepath.Join(t.TempDir(), "load.go")
	err := os.WriteFile(sourceFile, []byte(`package test

func Load(id string) (err error) {
	defer errs.WrapWithFuncParams(&err)

	return nil
}
`), 0600)
	require.NoError(t, err)

	t.Run("creates missing output directory", func(t *testing.T) {
		outDir := filepath.Join(t.TempDir(), "missing", "out")
		var verbose strings.Builder

		err := Replace(sourceFile, outDir, false, false, false, &verbose)
		require.NoError(t, err)

		outFile := filepath.Join(outDir, "load.go")
		assertWrapAndErrsImport(t, outFile, "defer errs.WrapWithFuncParams(&err, id)")
		assert.Contains(t, verbose.String(), "wrote: "+outFile)
	})

	t.Run("write error", func(t *testing.T) {
		// A regular file in place of the parent directory makes writing fail
		parent := filepath.Join(t.TempDir(), "file")
		err := os.WriteFile(parent, nil, 0600)
		require.NoError(t, err)
		outFile := filepath.Join(parent, "load.go")

		err = Replace(sourceFile, outFile, false, false, false, nil)
		require.Error(t, err)
	})
}

// TestRemoveInPlaceDropsUnusedErrsImport verifies that removing wrap
// statements in place also removes the go-errs import once nothing uses it,
// so the rewritten file still compiles, and keeps the import while other
// code still uses it.
func TestRemoveInPlaceDropsUnusedErrsImport(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name: "unused after removal",
			input: `package test

import "github.com/domonda/go-errs"

func Load(id string) (err error) {
	defer errs.WrapWithFuncParams(&err, id)

	return nil
}
`,
			want: `package test

func Load(id string) (err error) {

	return nil
}
`,
		},
		{
			name: "aliased unused after removal",
			input: `package test

import e "github.com/domonda/go-errs"

func Load(id string) (err error) {
	defer e.WrapWithFuncParams(&err, id)

	return nil
}
`,
			want: `package test

func Load(id string) (err error) {

	return nil
}
`,
		},
		{
			name: "aliased still used",
			input: `package test

import e "github.com/domonda/go-errs"

var errNotFound = e.New("not found")

func Load(id string) (err error) {
	defer e.WrapWithFuncParams(&err, id)

	return errNotFound
}
`,
			want: `package test

import e "github.com/domonda/go-errs"

var errNotFound = e.New("not found")

func Load(id string) (err error) {

	return errNotFound
}
`,
		},
		{
			name: "dot import kept",
			input: `package test

import (
	. "github.com/domonda/go-errs"
	"github.com/domonda/go-errs"
)

var errNotFound = New("not found")

func Load(id string) (err error) {
	defer errs.WrapWithFuncParams(&err, id)

	return errNotFound
}
`,
			want: `package test

import (
	. "github.com/domonda/go-errs"
)

var errNotFound = New("not found")

func Load(id string) (err error) {

	return errNotFound
}
`,
		},
		{
			name: "still used",
			input: `package test

import "github.com/domonda/go-errs"

func Load(id string) (err error) {
	defer errs.WrapWithFuncParams(&err, id)

	return errs.New("not implemented")
}
`,
			want: `package test

import "github.com/domonda/go-errs"

func Load(id string) (err error) {

	return errs.New("not implemented")
}
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "load.go")
			err := os.WriteFile(file, []byte(tt.input), 0600)
			require.NoError(t, err)
			var verbose strings.Builder

			err = Remove(file, "", false, false, &verbose)
			require.NoError(t, err)

			output, err := os.ReadFile(file)
			require.NoError(t, err)
			assert.Equal(t, tt.want, string(output))
			assert.Contains(t, verbose.String(), "changed file: "+file)
		})
	}

	t.Run("write error", func(t *testing.T) {
		if os.Geteuid() == 0 {
			t.Skip("root can write read-only files")
		}
		const input = `package test

import "github.com/domonda/go-errs"

func Load(id string) (err error) {
	defer errs.WrapWithFuncParams(&err, id)

	return nil
}
`
		file := filepath.Join(t.TempDir(), "load.go")
		err := os.WriteFile(file, []byte(input), 0400)
		require.NoError(t, err)

		err = Remove(file, "", false, false, nil)
		require.Error(t, err)

		output, err := os.ReadFile(file)
		require.NoError(t, err)
		assert.Equal(t, input, string(output))
	})
}

// TestRewriteUsesErrsImportName verifies that generated statements use the
// name under which a file imports go-errs, including for errs.KeepSecret,
// instead of adding a second unnamed import that leaves the named one unused.
func TestRewriteUsesErrsImportName(t *testing.T) {
	tests := []struct {
		name    string
		rewrite func(sourcePath string) error
		input   string
		want    string
	}{
		{
			name: "replace",
			rewrite: func(sourcePath string) error {
				return Replace(sourcePath, "", false, false, false, nil)
			},
			input: `package test

import e "github.com/domonda/go-errs"

func Login(user, password string) (err error) {
	defer e.WrapWithFuncParams(&err, user, e.KeepSecret(password))

	return nil
}
`,
			want: `package test

import e "github.com/domonda/go-errs"

func Login(user, password string) (err error) {
	defer e.WrapWithFuncParams(&err, user, e.KeepSecret(password))

	return nil
}
`,
		},
		{
			name: "replace outdated",
			rewrite: func(sourcePath string) error {
				return Replace(sourcePath, "", false, true, false, nil)
			},
			input: `package test

import e "github.com/domonda/go-errs"

func Load(id string) (err error) {
	defer e.WrapWithFuncParams(&err)

	return nil
}
`,
			want: `package test

import e "github.com/domonda/go-errs"

func Load(id string) (err error) {
	defer e.WrapWith1FuncParam(&err, id)

	return nil
}
`,
		},
		{
			name: "insert",
			rewrite: func(sourcePath string) error {
				return Insert(sourcePath, "", false, true, false, nil)
			},
			input: `package test

import e "github.com/domonda/go-errs"

var errNotFound = e.New("not found")

func Load(id string) (err error) {
	return errNotFound
}
`,
			want: `package test

import e "github.com/domonda/go-errs"

var errNotFound = e.New("not found")

func Load(id string) (err error) {
	defer e.WrapWith1FuncParam(&err, id)

	return errNotFound
}
`,
		},
		{
			name: "replace marker comment",
			rewrite: func(sourcePath string) error {
				return Replace(sourcePath, "", false, true, false, nil)
			},
			input: `package test

import e "github.com/domonda/go-errs"

var errNotFound = e.New("not found")

func Load(id string) (err error) {
	//#wrap-result-err

	return errNotFound
}
`,
			want: `package test

import e "github.com/domonda/go-errs"

var errNotFound = e.New("not found")

func Load(id string) (err error) {
	defer e.WrapWith1FuncParam(&err, id)

	return errNotFound
}
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "load.go")
			err := os.WriteFile(file, []byte(tt.input), 0600)
			require.NoError(t, err)

			err = tt.rewrite(file)
			require.NoError(t, err)

			output, err := os.ReadFile(file)
			require.NoError(t, err)
			assert.Equal(t, tt.want, string(output))
		})
	}
}

// TestRewriteToOutputDirectoryCopiesTree verifies that the output directory
// is a copy of the source directory tree in which only the rewritten files
// differ: unchanged Go files and the Go files the go command ignores,
// like test files, files with custom build tags, files in testdata, vendor,
// _ or . directories and nested modules are copied as they are,
// and without recursion only the files directly in the source directory
// are copied and rewritten.
func TestRewriteToOutputDirectoryCopiesTree(t *testing.T) {
	const loadSource = `package test

func Load(id string) (err error) {
	return nil
}
`
	const unchangedSource = `package test

func Unchanged() int { return 1 }
`
	// Files the go command ignores are copied without changes
	ignoredFiles := map[string]string{
		"load_test.go":          loadSource,
		"tagged.go":             "//go:build customtag\n\n" + loadSource,
		"_ignored.go":           loadSource,
		"testdata/fixture/f.go": loadSource,
		"vendor/v/v.go":         loadSource,
		".hidden/h.go":          loadSource,
		"_dir/d.go":             loadSource,
		"nested/go.mod":         "module example.com/nested\n",
		"nested/n.go":           loadSource,
	}
	sourceDir := t.TempDir()
	files := map[string]string{
		// Only nested modules of a module are skipped
		"go.mod":                  "module example.com/test\n",
		"load.go":                 loadSource,
		"unchanged.go":            unchangedSource,
		"sub/load.go":             loadSource,
		"testdata/fixture/f.json": "{}\n",
	}
	for relPath, content := range ignoredFiles {
		files[relPath] = content
	}
	for relPath, content := range files {
		path := filepath.Join(sourceDir, filepath.FromSlash(relPath))
		err := os.MkdirAll(filepath.Dir(path), 0750)
		require.NoError(t, err)
		err = os.WriteFile(path, []byte(content), 0600)
		require.NoError(t, err)
	}
	const wantLoad = `package test

import "github.com/domonda/go-errs"

func Load(id string) (err error) {
	defer errs.WrapWith1FuncParam(&err, id)

	return nil
}
`
	readOutput := func(t *testing.T, outDir, relPath string) string {
		t.Helper()
		output, err := os.ReadFile(filepath.Join(outDir, filepath.FromSlash(relPath)))
		require.NoError(t, err)
		return string(output)
	}

	t.Run("recursive", func(t *testing.T) {
		outDir := t.TempDir()

		err := Insert(sourceDir, outDir, true, true, false, nil)
		require.NoError(t, err)

		assert.Equal(t, wantLoad, readOutput(t, outDir, "load.go"))
		assert.Equal(t, unchangedSource, readOutput(t, outDir, "unchanged.go"))
		assert.Equal(t, wantLoad, readOutput(t, outDir, "sub/load.go"))
		assert.Equal(t, "{}\n", readOutput(t, outDir, "testdata/fixture/f.json"))
		for relPath, content := range ignoredFiles {
			assert.Equal(t, content, readOutput(t, outDir, relPath), relPath)
		}
	})

	t.Run("not recursive", func(t *testing.T) {
		outDir := t.TempDir()

		err := Insert(sourceDir, outDir, false, true, false, nil)
		require.NoError(t, err)

		assert.Equal(t, wantLoad, readOutput(t, outDir, "load.go"))
		assert.Equal(t, unchangedSource, readOutput(t, outDir, "unchanged.go"))
		assert.Equal(t, ignoredFiles["load_test.go"], readOutput(t, outDir, "load_test.go"))
		assert.NoDirExists(t, filepath.Join(outDir, "sub"))
		assert.NoDirExists(t, filepath.Join(outDir, "testdata"))
	})
}

// TestRewriteFallsBackToUnnamedErrsImport verifies that a function declaring
// the name of an aliased go-errs import, which shadows the package there,
// gets a statement with the unnamed import, which is added.
func TestRewriteFallsBackToUnnamedErrsImport(t *testing.T) {
	const input = `package test

import e "github.com/domonda/go-errs"

var errNotFound = e.New("not found")

type T struct{}

func Handle(e string) (err error) {
	return errNotFound
}

func (e *T) Method() (err error) {
	return errNotFound
}
`
	const want = `package test

import (
	"github.com/domonda/go-errs"
	e "github.com/domonda/go-errs"
)

var errNotFound = e.New("not found")

type T struct{}

func Handle(e string) (err error) {
	defer errs.WrapWith1FuncParam(&err, e)

	return errNotFound
}

func (e *T) Method() (err error) {
	defer errs.WrapWith0FuncParams(&err)

	return errNotFound
}
`
	file := filepath.Join(t.TempDir(), "load.go")
	err := os.WriteFile(file, []byte(input), 0600)
	require.NoError(t, err)

	err = Insert(file, "", false, true, false, nil)
	require.NoError(t, err)

	output, err := os.ReadFile(file)
	require.NoError(t, err)
	assert.Equal(t, want, string(output))
}

// TestRemoveKeepsOtherImports verifies that remove only drops the go-errs
// import, keeps the import groups and other unused imports, and treats
// a local variable named errs as not using the package.
func TestRemoveKeepsOtherImports(t *testing.T) {
	const input = `package test

import (
	"fmt"
	"github.com/domonda/go-errs"
	"strings"
)

type collector struct{ n int }

func Load(id string) (err error) {
	defer errs.WrapWithFuncParams(&err, id)

	errs := collector{}
	fmt.Println(errs.n)
	return nil
}
`
	const want = `package test

import (
	"fmt"
	"strings"
)

type collector struct{ n int }

func Load(id string) (err error) {

	errs := collector{}
	fmt.Println(errs.n)
	return nil
}
`
	file := filepath.Join(t.TempDir(), "load.go")
	err := os.WriteFile(file, []byte(input), 0600)
	require.NoError(t, err)

	err = Remove(file, "", false, false, nil)
	require.NoError(t, err)

	output, err := os.ReadFile(file)
	require.NoError(t, err)
	assert.Equal(t, want, string(output))
}

// TestRewriteKeepsCRLFLineEndings verifies that rewritten files keep
// the CRLF line endings of their source, in place and with -out.
func TestRewriteKeepsCRLFLineEndings(t *testing.T) {
	const input = "package test\r\n\r\nimport \"github.com/domonda/go-errs\"\r\n\r\nfunc Load(id string) (err error) {\r\n\tdefer errs.WrapWithFuncParams(&err, id)\r\n\r\n\treturn nil\r\n}\r\n"
	const want = "package test\r\n\r\nfunc Load(id string) (err error) {\r\n\r\n\treturn nil\r\n}\r\n"

	t.Run("in place", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "load.go")
		err := os.WriteFile(file, []byte(input), 0600)
		require.NoError(t, err)

		err = Remove(file, "", false, false, nil)
		require.NoError(t, err)

		output, err := os.ReadFile(file)
		require.NoError(t, err)
		assert.Equal(t, want, string(output))
	})

	t.Run("output directory", func(t *testing.T) {
		sourceDir := t.TempDir()
		err := os.WriteFile(filepath.Join(sourceDir, "load.go"), []byte(input), 0600)
		require.NoError(t, err)
		outDir := t.TempDir()

		err = Remove(sourceDir, outDir, false, false, nil)
		require.NoError(t, err)

		output, err := os.ReadFile(filepath.Join(outDir, "load.go"))
		require.NoError(t, err)
		assert.Equal(t, want, string(output))
	})
}

// TestRewriteOutputDirectoryAndSourceOverlap verifies that an output path
// that is the source directory, is within it or contains it is rejected
// before anything is written, because the output would overwrite source
// files, also for another spelling of the same directory.
func TestRewriteOutputDirectoryAndSourceOverlap(t *testing.T) {
	const loadSource = `package test

func Load(id string) (err error) {
	return nil
}
`
	parentDir := t.TempDir()
	sourceDir := filepath.Join(parentDir, "src")
	err := os.MkdirAll(filepath.Join(sourceDir, "sub"), 0750)
	require.NoError(t, err)
	for _, relPath := range []string{"load.go", filepath.Join("sub", "load.go")} {
		err = os.WriteFile(filepath.Join(sourceDir, relPath), []byte(loadSource), 0600)
		require.NoError(t, err)
	}
	linkToSource := filepath.Join(parentDir, "link")
	err = os.Symlink(sourceDir, linkToSource)
	require.NoError(t, err)

	outPaths := map[string]string{
		"source":                 sourceDir,
		"existing sub-directory": filepath.Join(sourceDir, "sub"),
		"new sub-directory":      filepath.Join(sourceDir, "out"),
		"parent":                 parentDir,
		"through link":           filepath.Join(linkToSource, "out"),
	}
	// On a case-insensitive file system, like the macOS default,
	// another letter case spells the same directory
	upperSourceDir := filepath.Join(parentDir, "SRC")
	if _, err := os.Stat(upperSourceDir); err == nil {
		outPaths["other letter case"] = upperSourceDir
	}
	for name, outPath := range outPaths {
		t.Run(name, func(t *testing.T) {
			for _, recursive := range []bool{true, false} {
				err := Insert(sourceDir, outPath, recursive, true, false, nil)
				require.ErrorContains(t, err, "must not be the source directory")
			}
			for _, relPath := range []string{"load.go", filepath.Join("sub", "load.go")} {
				output, err := os.ReadFile(filepath.Join(sourceDir, relPath))
				require.NoError(t, err)
				assert.Equal(t, loadSource, string(output), relPath)
			}
			assert.NoDirExists(t, filepath.Join(sourceDir, "out"))
			assert.NoFileExists(t, filepath.Join(parentDir, "load.go"))
		})
	}
}

// TestRewriteToOutputDirectoryReplacesLinks verifies that files in the output
// directory are replaced, not written through, so that an output tree of hard
// or symbolic links to the source files, like a copy made with cp -al,
// doesn't change the source files.
func TestRewriteToOutputDirectoryReplacesLinks(t *testing.T) {
	const loadSource = `package test

func Load(id string) (err error) {
	return nil
}
`
	sourceDir := t.TempDir()
	err := os.WriteFile(filepath.Join(sourceDir, "load.go"), []byte(loadSource), 0600)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(sourceDir, "notes.txt"), []byte("notes\n"), 0600)
	require.NoError(t, err)
	outDir := t.TempDir()
	err = os.Link(filepath.Join(sourceDir, "load.go"), filepath.Join(outDir, "load.go"))
	require.NoError(t, err)
	err = os.Symlink(filepath.Join(sourceDir, "notes.txt"), filepath.Join(outDir, "notes.txt"))
	require.NoError(t, err)

	err = Insert(sourceDir, outDir, false, true, false, nil)
	require.NoError(t, err)

	assertWrapAndErrsImport(t, filepath.Join(outDir, "load.go"), "defer errs.WrapWith1FuncParam(&err, id)")
	source, err := os.ReadFile(filepath.Join(sourceDir, "load.go"))
	require.NoError(t, err)
	assert.Equal(t, loadSource, string(source))
	info, err := os.Lstat(filepath.Join(outDir, "notes.txt"))
	require.NoError(t, err)
	assert.True(t, info.Mode().IsRegular(), "notes.txt link replaced by a regular file")
}

// TestRewriteToOutputDirectoryFromLinkedSource verifies that a source
// directory passed as symbolic link is copied and rewritten.
func TestRewriteToOutputDirectoryFromLinkedSource(t *testing.T) {
	realDir := filepath.Join(t.TempDir(), "real")
	err := os.MkdirAll(realDir, 0750)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(realDir, "load.go"), []byte("package test\n\nfunc Load(id string) (err error) {\n\treturn nil\n}\n"), 0600)
	require.NoError(t, err)
	linkDir := filepath.Join(t.TempDir(), "link")
	err = os.Symlink(realDir, linkDir)
	require.NoError(t, err)
	outDir := t.TempDir()

	err = Insert(linkDir, outDir, true, true, false, nil)
	require.NoError(t, err)

	assertWrapAndErrsImport(t, filepath.Join(outDir, "load.go"), "defer errs.WrapWith1FuncParam(&err, id)")
}

// TestRewriteToOutputDirectoryWritesNothingOnError verifies that no output
// is written if rewriting any file fails, even for packages rewritten before.
func TestRewriteToOutputDirectoryWritesNothingOnError(t *testing.T) {
	sourceDir := t.TempDir()
	for relPath, content := range map[string]string{
		"a/a.go": "package a\n\nfunc A(id string) (err error) {\n\t//#wrap-result-err\n\n\treturn nil\n}\n",
		// The marker after a statement on the same line becomes invalid code
		"b/b.go": "package b\n\nfunc B(id string) (err error) {\n\tx := 1 //#wrap-result-err\n\t_ = x\n\treturn nil\n}\n",
	} {
		path := filepath.Join(sourceDir, filepath.FromSlash(relPath))
		err := os.MkdirAll(filepath.Dir(path), 0750)
		require.NoError(t, err)
		err = os.WriteFile(path, []byte(content), 0600)
		require.NoError(t, err)
	}
	outDir := filepath.Join(t.TempDir(), "out")

	err := Replace(sourceDir, outDir, true, true, false, nil)
	require.Error(t, err)
	assert.NoDirExists(t, outDir)
}

// TestRewriteToOutputDirectorySkipsSpecialFiles verifies how the output
// directory treats files that are not regular: a symbolic link to a file
// is written as regular file without changing the linked file,
// other symbolic links are skipped, like version control metadata.
func TestRewriteToOutputDirectorySkipsSpecialFiles(t *testing.T) {
	const loadSource = `package test

func Load(id string) (err error) {
	return nil
}
`
	rootDir := t.TempDir()
	sharedFile := filepath.Join(rootDir, "shared", "shared.go")
	err := os.MkdirAll(filepath.Dir(sharedFile), 0750)
	require.NoError(t, err)
	err = os.WriteFile(sharedFile, []byte(loadSource), 0600)
	require.NoError(t, err)
	sourceDir := filepath.Join(rootDir, "src")
	err = os.MkdirAll(filepath.Join(sourceDir, ".git"), 0750)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(sourceDir, ".git", "HEAD"), []byte("ref: refs/heads/main\n"), 0600)
	require.NoError(t, err)
	err = os.Symlink(sharedFile, filepath.Join(sourceDir, "link.go"))
	require.NoError(t, err)
	err = os.Symlink(filepath.Dir(sharedFile), filepath.Join(sourceDir, "dirlink"))
	require.NoError(t, err)
	err = os.Symlink(filepath.Join(rootDir, "missing.txt"), filepath.Join(sourceDir, "dangling.txt"))
	require.NoError(t, err)
	// A git worktree has a .git file instead of a directory
	err = os.MkdirAll(filepath.Join(sourceDir, "worktree"), 0750)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(sourceDir, "worktree", ".git"), []byte("gitdir: /elsewhere\n"), 0600)
	require.NoError(t, err)
	// Unix socket paths are limited to about 100 bytes, too short for t.TempDir on macOS
	socketDir, err := os.MkdirTemp("", "gew")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(socketDir) })
	socketSourceDir := filepath.Join(socketDir, "src")
	err = os.Symlink(sourceDir, socketSourceDir)
	require.NoError(t, err)
	listener, err := net.Listen("unix", filepath.Join(socketSourceDir, "s"))
	if err != nil {
		t.Logf("no special file to test: %s", err)
	} else {
		t.Cleanup(func() { listener.Close() })
	}
	outDir := t.TempDir()
	var verbose strings.Builder

	err = Insert(sourceDir, outDir, true, true, false, &verbose)
	require.NoError(t, err)

	outFile := filepath.Join(outDir, "link.go")
	assertWrapAndErrsImport(t, outFile, "defer errs.WrapWith1FuncParam(&err, id)")
	info, err := os.Lstat(outFile)
	require.NoError(t, err)
	assert.True(t, info.Mode().IsRegular(), "link.go is written as regular file")
	shared, err := os.ReadFile(sharedFile)
	require.NoError(t, err)
	assert.Equal(t, loadSource, string(shared), "linked file is unchanged")
	assert.NoFileExists(t, filepath.Join(outDir, "dirlink"))
	assert.NoFileExists(t, filepath.Join(outDir, "dangling.txt"))
	assert.Contains(t, verbose.String(), "skipping symbolic link: "+filepath.Join(sourceDir, "dangling.txt"))
	assert.NoDirExists(t, filepath.Join(outDir, ".git"))
	assert.NoFileExists(t, filepath.Join(outDir, "worktree", ".git"))
	if listener != nil {
		assert.NoFileExists(t, filepath.Join(outDir, "s"))
		assert.Contains(t, verbose.String(), "skipping irregular file: "+filepath.Join(sourceDir, "s"))
	}
}

// TestRewriteSkipsTestFilePath verifies that a test file passed as path
// is rejected instead of being processed.
func TestRewriteSkipsTestFilePath(t *testing.T) {
	file := filepath.Join(t.TempDir(), "load_test.go")
	err := os.WriteFile(file, []byte("package test\n\nfunc Load(id string) (err error) {\n\treturn nil\n}\n"), 0600)
	require.NoError(t, err)
	outDir := t.TempDir()

	err = Insert(file, outDir, false, true, false, nil)
	require.ErrorContains(t, err, "is not rewritten")
	assert.NoFileExists(t, filepath.Join(outDir, "load_test.go"))
}

// TestRewriteWithOtherErrsPackage verifies that a file importing another
// package as errs gets go-errs imported as goerrs instead of a second errs
// import, and that the defer statements calling the other package
// are not taken for go-errs statements.
func TestRewriteWithOtherErrsPackage(t *testing.T) {
	const input = `package test

import "example.com/proj/pkg/errs"

func Load(id string) (err error) {
	defer errs.WrapIfFailed(&err)

	return nil
}
`
	t.Run("insert", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "load.go")
		err := os.WriteFile(file, []byte(input), 0600)
		require.NoError(t, err)

		err = Insert(file, "", false, true, false, nil)
		require.NoError(t, err)

		output, err := os.ReadFile(file)
		require.NoError(t, err)
		assert.Equal(t, `package test

import (
	"example.com/proj/pkg/errs"
	goerrs "github.com/domonda/go-errs"
)

func Load(id string) (err error) {
	defer goerrs.WrapWith1FuncParam(&err, id)

	defer errs.WrapIfFailed(&err)

	return nil
}
`, string(output))
	})

	t.Run("remove and replace", func(t *testing.T) {
		file := filepath.Join(t.TempDir(), "load.go")
		err := os.WriteFile(file, []byte(input), 0600)
		require.NoError(t, err)

		err = Remove(file, "", false, true, nil)
		require.NoError(t, err)
		err = Replace(file, "", false, false, true, nil)
		require.NoError(t, err)
	})
}

// TestRewriteAvoidsShadowedErrsNames verifies that generated statements never
// reference go-errs by a name that a parameter or local declaration shadows,
// importing go-errs under a free name instead, and that a method call on such
// a variable is not taken for a go-errs statement.
func TestRewriteAvoidsShadowedErrsNames(t *testing.T) {
	tests := []struct {
		name    string
		rewrite func(sourcePath string) error
		input   string
		want    string
	}{
		{
			name: "marker in function shadowing alias",
			rewrite: func(sourcePath string) error {
				return Replace(sourcePath, "", false, true, false, nil)
			},
			input: `package test

import e "github.com/domonda/go-errs"

var errNotFound = e.New("not found")

func Handle(e string) (err error) {
	//#wrap-result-err

	return errNotFound
}
`,
			want: `package test

import (
	"github.com/domonda/go-errs"
	e "github.com/domonda/go-errs"
)

var errNotFound = e.New("not found")

func Handle(e string) (err error) {
	defer errs.WrapWith1FuncParam(&err, e)

	return errNotFound
}
`,
		},
		{
			name: "parameter named errs",
			rewrite: func(sourcePath string) error {
				return Insert(sourcePath, "", false, true, false, nil)
			},
			input: `package test

import "errors"

func Combine(errs ...error) (err error) {
	return errors.Join(errs...)
}
`,
			want: `package test

import (
	"errors"

	goerrs "github.com/domonda/go-errs"
)

func Combine(errs ...error) (err error) {
	defer goerrs.WrapWith1FuncParam(&err, errs)

	return errors.Join(errs...)
}
`,
		},
		{
			name: "locals shadowing errs and goerrs",
			rewrite: func(sourcePath string) error {
				return Insert(sourcePath, "", false, true, false, nil)
			},
			input: `package test

func Outer() {
	errs, goerrs := 1, 2
	f := func(id string) (err error) {
		return nil
	}
	_, _, _ = errs, goerrs, f
}
`,
			want: `package test

import goerrs2 "github.com/domonda/go-errs"

func Outer() {
	errs, goerrs := 1, 2
	f := func(id string) (err error) {
		defer goerrs2.WrapWith1FuncParam(&err, id)

		return nil
	}
	_, _, _ = errs, goerrs, f
}
`,
		},
		{
			name: "fallback name used by another import",
			rewrite: func(sourcePath string) error {
				return Insert(sourcePath, "", false, true, false, nil)
			},
			input: `package test

import goerrs "github.com/go-errors/errors"

func Combine(errs ...error) (err error) {
	return goerrs.New("combined")
}
`,
			want: `package test

import (
	goerrs2 "github.com/domonda/go-errs"
	goerrs "github.com/go-errors/errors"
)

func Combine(errs ...error) (err error) {
	defer goerrs2.WrapWith1FuncParam(&err, errs)

	return goerrs.New("combined")
}
`,
		},
		{
			name: "method call on shadowing parameter",
			rewrite: func(sourcePath string) error {
				return Remove(sourcePath, "", false, false, nil)
			},
			input: `package test

import e "github.com/domonda/go-errs"

var errNotFound = e.New("not found")

type Entry struct{}

func (*Entry) WrapUp() {}

func Close(e *Entry) (err error) {
	defer e.WrapUp()

	return errNotFound
}
`,
			want: `package test

import e "github.com/domonda/go-errs"

var errNotFound = e.New("not found")

type Entry struct{}

func (*Entry) WrapUp() {}

func Close(e *Entry) (err error) {
	defer e.WrapUp()

	return errNotFound
}
`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "load.go")
			err := os.WriteFile(file, []byte(tt.input), 0600)
			require.NoError(t, err)

			err = tt.rewrite(file)
			require.NoError(t, err)

			output, err := os.ReadFile(file)
			require.NoError(t, err)
			assert.Equal(t, tt.want, string(output))
		})
	}
}

// TestRewriteToOutputDirectoryStaysWithin verifies that a symbolic link to a
// directory in an existing output directory can't make the output overwrite
// files outside of it, like the source files.
func TestRewriteToOutputDirectoryStaysWithin(t *testing.T) {
	const loadSource = `package test

func Load(id string) (err error) {
	return nil
}
`
	sourceDir := t.TempDir()
	err := os.MkdirAll(filepath.Join(sourceDir, "sub"), 0750)
	require.NoError(t, err)
	for _, relPath := range []string{"load.go", filepath.Join("sub", "load.go")} {
		err = os.WriteFile(filepath.Join(sourceDir, relPath), []byte(loadSource), 0600)
		require.NoError(t, err)
	}
	elsewhereDir := t.TempDir()
	err = os.WriteFile(filepath.Join(elsewhereDir, "load.go"), []byte(loadSource), 0600)
	require.NoError(t, err)

	for name, linkTarget := range map[string]string{
		"link to source":    sourceDir,
		"link to elsewhere": elsewhereDir,
	} {
		t.Run(name, func(t *testing.T) {
			outDir := t.TempDir()
			err := os.Symlink(linkTarget, filepath.Join(outDir, "sub"))
			require.NoError(t, err)

			err = Insert(sourceDir, outDir, true, true, false, nil)
			require.Error(t, err)

			for _, path := range []string{
				filepath.Join(sourceDir, "load.go"),
				filepath.Join(sourceDir, "sub", "load.go"),
				filepath.Join(elsewhereDir, "load.go"),
			} {
				output, err := os.ReadFile(path)
				require.NoError(t, err)
				assert.Equal(t, loadSource, string(output), path)
			}
		})
	}
}

// TestRewriteSingleFileOutputKeepsPermissions verifies that a single file
// written to an output path gets the permissions of the source file.
func TestRewriteSingleFileOutputKeepsPermissions(t *testing.T) {
	sourceFile := filepath.Join(t.TempDir(), "load.go")
	err := os.WriteFile(sourceFile, []byte("package test\n\nfunc Load(id string) (err error) {\n\treturn nil\n}\n"), 0600)
	require.NoError(t, err)
	err = os.Chmod(sourceFile, 0644)
	require.NoError(t, err)
	outFile := filepath.Join(t.TempDir(), "load.go")

	err = Insert(sourceFile, outFile, false, true, false, nil)
	require.NoError(t, err)

	info, err := os.Stat(outFile)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0644), info.Mode().Perm())
}
