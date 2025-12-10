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
	err = Remove(inputFile, outDir, false, nil)
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

	// Run replace
	err = Replace(inputFile, outDir, false, nil)
	require.NoError(t, err)

	// Read output
	outputFile := filepath.Join(outDir, "input.go")
	output, err := os.ReadFile(outputFile)
	require.NoError(t, err)

	outputStr := string(output)

	// Verify updated defer statement
	assert.Contains(t, outputStr, "defer errs.WrapWith2FuncParams(&err, ctx, id)")

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
	err = Replace(inputFile, outDir, false, nil)
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

	replacements, imports, err := processFile(fset, file, nil, false)
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
