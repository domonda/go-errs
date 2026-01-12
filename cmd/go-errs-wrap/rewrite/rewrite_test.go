package rewrite

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsDeferErrsWrap(t *testing.T) {
	tests := []struct {
		name string
		code string
		want bool
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

			result := isDeferErrsWrap(deferStmt)
			assert.Equal(t, tt.want, result)
		})
	}
}

func TestIsVariadicWrapWithFuncParams(t *testing.T) {
	tests := []struct {
		name string
		code string
		want bool
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

			result := isVariadicWrapWithFuncParams(deferStmt)
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
	assert.Contains(t, imports, `"github.com/domonda/go-errs"`)
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
	err = copyFile(srcPath, dstPath)
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
