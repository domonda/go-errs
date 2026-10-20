package rewrite

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractFuncInfo(t *testing.T) {
	tests := []struct {
		name            string
		code            string
		wantFuncName    string
		wantParams      []string
		wantErrorResult string
	}{
		{
			name: "simple function with params and error",
			code: `package test
func Foo(a string, b int) (result string, err error) { return }`,
			wantFuncName:    "Foo",
			wantParams:      []string{"a", "b"},
			wantErrorResult: "err",
		},
		{
			name: "function with no params",
			code: `package test
func Bar() (err error) { return }`,
			wantFuncName:    "Bar",
			wantParams:      nil,
			wantErrorResult: "err",
		},
		{
			name: "function with blank param",
			code: `package test
func Baz(_ string, x int) error { return nil }`,
			wantFuncName:    "Baz",
			wantParams:      []string{"x"},
			wantErrorResult: "",
		},
		{
			name: "function with unnamed error result",
			code: `package test
func NoNamedErr(x int) error { return nil }`,
			wantFuncName:    "NoNamedErr",
			wantParams:      []string{"x"},
			wantErrorResult: "",
		},
		{
			name: "function with multiple error results uses last",
			code: `package test
func MultiErr(x int) (err1 error, err2 error) { return }`,
			wantFuncName:    "MultiErr",
			wantParams:      []string{"x"},
			wantErrorResult: "err2",
		},
		{
			name: "function with context and multiple params",
			code: `package test
import "context"
func Process(ctx context.Context, id string, count int) (err error) { return }`,
			wantFuncName:    "Process",
			wantParams:      []string{"ctx", "id", "count"},
			wantErrorResult: "err",
		},
		{
			name: "function with variadic param",
			code: `package test
func Variadic(format string, args ...any) (err error) { return }`,
			wantFuncName:    "Variadic",
			wantParams:      []string{"format", "args"},
			wantErrorResult: "err",
		},
		{
			name: "function with grouped params",
			code: `package test
func Grouped(a, b, c string) (err error) { return }`,
			wantFuncName:    "Grouped",
			wantParams:      []string{"a", "b", "c"},
			wantErrorResult: "err",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "test.go", tt.code, parser.ParseComments)
			require.NoError(t, err)

			var funcDecl *ast.FuncDecl
			for _, decl := range file.Decls {
				if fd, ok := decl.(*ast.FuncDecl); ok {
					funcDecl = fd
					break
				}
			}
			require.NotNil(t, funcDecl, "no function declaration found")

			ctx := extractFuncInfo(funcDecl.Type, funcDecl.Name.Name, funcDecl.Pos(), funcDecl.End())

			assert.Equal(t, tt.wantFuncName, ctx.funcName)
			assert.Equal(t, tt.wantParams, ctx.paramNames)
			assert.Equal(t, tt.wantErrorResult, ctx.errorResultName)
		})
	}
}

func TestIsErrorType(t *testing.T) {
	tests := []struct {
		name string
		code string
		want bool
	}{
		{
			name: "error type",
			code: "error",
			want: true,
		},
		{
			name: "string type",
			code: "string",
			want: false,
		},
		{
			name: "int type",
			code: "int",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr, err := parser.ParseExpr(tt.code)
			require.NoError(t, err)
			assert.Equal(t, tt.want, isErrorType(expr))
		})
	}
}

func TestFindEnclosingFuncForPos(t *testing.T) {
	code := `package test

func Outer(a int) (err error) {
	// comment in outer
	inner := func(b string) (innerErr error) {
		// comment in inner
		return nil
	}
	_ = inner
	return nil
}

func Another(x string) error {
	return nil
}
`

	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", code, parser.ParseComments)
	require.NoError(t, err)

	// Find position of "comment in outer"
	var outerCommentPos token.Pos
	var innerCommentPos token.Pos
	for _, cg := range file.Comments {
		for _, c := range cg.List {
			if c.Text == "// comment in outer" {
				outerCommentPos = c.Pos()
			}
			if c.Text == "// comment in inner" {
				innerCommentPos = c.Pos()
			}
		}
	}

	require.NotEqual(t, token.NoPos, outerCommentPos, "outer comment not found")
	require.NotEqual(t, token.NoPos, innerCommentPos, "inner comment not found")

	// Test outer function context
	outerCtx := findEnclosingFuncForPos(file, outerCommentPos)
	require.NotNil(t, outerCtx)
	assert.Equal(t, "Outer", outerCtx.funcName)
	assert.Equal(t, []string{"a"}, outerCtx.paramNames)
	assert.Equal(t, "err", outerCtx.errorResultName)

	// Test inner function context (should find the anonymous function)
	innerCtx := findEnclosingFuncForPos(file, innerCommentPos)
	require.NotNil(t, innerCtx)
	assert.Equal(t, "(anonymous)", innerCtx.funcName)
	assert.Equal(t, []string{"b"}, innerCtx.paramNames)
	assert.Equal(t, "innerErr", innerCtx.errorResultName)
}

func TestErrsImportName(t *testing.T) {
	tests := []struct {
		name    string
		imports string
		want    string
	}{
		{name: "no import", imports: `import "fmt"`, want: ""},
		{name: "unnamed", imports: `import "github.com/domonda/go-errs"`, want: "errs"},
		{name: "aliased", imports: `import e "github.com/domonda/go-errs"`, want: "e"},
		{
			name: "unnamed preferred over aliased",
			imports: `import (
	e "github.com/domonda/go-errs"
	"github.com/domonda/go-errs"
)`,
			want: "errs",
		},
		// Generated code can't reference a blank or dot import
		{name: "blank", imports: `import _ "github.com/domonda/go-errs"`, want: ""},
		{name: "dot", imports: `import . "github.com/domonda/go-errs"`, want: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := parser.ParseFile(token.NewFileSet(), "test.go", "package test\n\n"+tt.imports+"\n", parser.ImportsOnly)
			require.NoError(t, err)

			assert.Equal(t, tt.want, errsImportName(file))
		})
	}
}

func TestIsShadowedAt(t *testing.T) {
	const source = `package test

type T struct{}

func (e *T) Method() (err error) {
	return nil // method
}

func Generic[e any](x e) (err error) {
	return nil // generic
}

func Results() (e int, err error) {
	return 0, nil // results
}

func Outer(e string) (err error) {
	inner := func() (err error) {
		return nil // nested
	}
	return inner()
}

func Literal() (err error) {
	f := func(e string) (err error) {
		return nil // literal
	}
	return f("")
}

func Locals(x any) (err error) {
	var e int
	_ = e
	if v, ok := x.(int); ok {
		_ = v // if init
	}
	for k := range 3 {
		_ = k // range
	}
	switch t := x.(type) {
	case int:
		_ = t // type switch
	}
	f := func() (err error) {
		return nil // before own declaration
	}
	g := 1
	_, _ = f, g
	return nil // local var
}

func Unrelated(x string) (err error) {
	return nil // unrelated
}
`
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "test.go", source, parser.ParseComments)
	require.NoError(t, err)
	posOf := func(comment string) token.Pos {
		t.Helper()
		for _, group := range file.Comments {
			if group.Text() == comment+"\n" {
				return group.Pos()
			}
		}
		t.Fatalf("comment %q not found", comment)
		return token.NoPos
	}

	for _, tt := range []struct {
		comment string
		name    string
		want    bool
	}{
		{comment: "method", want: true},
		{comment: "generic", want: true},
		{comment: "results", want: true},
		{comment: "nested", want: true},
		{comment: "literal", want: true},
		{comment: "unrelated", want: false},
		{comment: "local var", want: true},
		{comment: "if init", name: "v", want: true},
		{comment: "range", name: "k", want: true},
		{comment: "type switch", name: "t", want: true},
		{comment: "before own declaration", name: "f", want: false},
		{comment: "before own declaration", name: "g", want: false},
		{comment: "local var", name: "g", want: true},
	} {
		name := tt.name
		if name == "" {
			name = "e"
		}
		t.Run(tt.comment+" "+name, func(t *testing.T) {
			assert.Equal(t, tt.want, isShadowedAt(file, posOf(tt.comment), name))
		})
	}
}

func TestImportsOtherErrsPackage(t *testing.T) {
	tests := []struct {
		name    string
		imports string
		want    bool
	}{
		{name: "go-errs only", imports: `import "github.com/domonda/go-errs"`, want: false},
		{name: "unnamed errs package", imports: `import "example.com/proj/pkg/errs"`, want: true},
		{name: "versioned errs package", imports: `import "example.com/errs/v2"`, want: true},
		{name: "go- prefix", imports: `import "example.com/go-errs"`, want: true},
		{name: "named errs", imports: `import errs "example.com/proj/pkg/errors"`, want: true},
		{name: "renamed errs package", imports: `import pkgerrs "example.com/proj/pkg/errs"`, want: false},
		{name: "standard errors", imports: `import "errors"`, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file, err := parser.ParseFile(token.NewFileSet(), "test.go", "package test\n\n"+tt.imports+"\n", parser.ImportsOnly)
			require.NoError(t, err)

			assert.Equal(t, tt.want, importsOtherErrsPackage(file))
		})
	}
}
