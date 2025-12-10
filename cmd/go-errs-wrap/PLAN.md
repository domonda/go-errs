# Implementation Plan for go-errs-wrap

## Overview

A Go executable that adds, replaces, or removes `defer errs.WrapWithFuncParams` statements in Go source files.

## Location

`/Users/erik/go/src/github.com/domonda/go-errs/cmd/go-errs-wrap/main.go`

(Already exists as empty stub)

## Commands

### `remove`
Removes all code lines containing:
- `defer errs.Wrap` (any variant)
- Lines that are exactly `//#wrap-result-err` (with optional leading/trailing whitespace)

### `replace`
Replaces lines containing either:
- `defer errs.Wrap` (any variant)
- Lines that are exactly `//#wrap-result-err` (with optional leading/trailing whitespace)

With a generated line:
```go
defer errs.WrapWithFuncParams(&<RESULT_ERR>, <FUNC_PARAMS>)
```

Where:
- `<RESULT_ERR>` is the name of the enclosing function's named error result
- `<FUNC_PARAMS>` is a comma-separated list of the function's parameter names (without types)

**Requirements:**
- The enclosing function MUST have a named result of type `error`
- If not, output an error message and continue with the next replacement

## Command Line Interface

```
go-errs-wrap <command> [options] <path>

Commands:
  remove   Remove all `defer errs.Wrap` or `//#wrap-result-err` lines
  replace  Replace `defer errs.Wrap` or `//#wrap-result-err` with generated code

Arguments:
  sourcePath Source directory or file path
           - If directory: recursively process all *.go files (excluding *_test.go)
           - If file: process only that file

Options:
  -out <path>   Output to different location instead of modifying source
                - If source is directory: create copy of directory structure
                - If source is file: write to specified file path that can be a directory that must exist, add the basename of the source file to the directory, or file path that ends with `.go` marking a filename directly
                - Non-Go files are copied unchanged
  -verbose      Print progress information
  -help         Show help message
```

## Implementation Structure

### Files

```
cmd/go-errs-wrap/
├── main.go              # Entry point, CLI parsing
└── rewrite/
    ├── rewrite.go       # Core Remove and Replace functions
    ├── funccontext.go   # Function context extraction
    └── generate.go      # Wrap statement generation
```

The `rewrite` subpackage contains all the core logic, while `main.go` handles CLI parsing and invokes the appropriate functions from the `rewrite` package.

### Dependencies

- `github.com/ungerik/go-astvisit` - For AST-based rewriting (same as go-enum)
- Standard library: `go/ast`, `go/token`, `go/parser`, `flag`, `os`, `path/filepath`, `io`, `io/fs`

### Core Logic

#### 1. AST Visitor Pattern

Use `astvisit.RewriteWithReplacements` to:
1. Parse Go files into AST
2. Find target statements (defer errs.Wrap or //#wrap-result-err comments)
3. Generate replacements
4. Apply replacements with proper formatting

#### 2. Finding Target Statements

For each file, traverse the AST looking for:

**Remove command targets:**
- `*ast.DeferStmt` where the call expression matches `errs.Wrap*` pattern
- `*ast.Comment` that is exactly `//#wrap-result-err` (trimmed of whitespace)

**Replace command targets:**
- Same as above (existing defer errs.Wrap statements)
- `*ast.Comment` containing `//#wrap-result-err`

#### 3. Finding Enclosing Function Context

When processing a target:
1. Walk up the AST to find the enclosing `*ast.FuncDecl` or `*ast.FuncLit`
2. Extract function parameters (names only, no types)
3. Find the named error result in the return type list
4. Validate that a named error result exists

#### 4. Generating Replacement Code

Format:
```go
defer errs.WrapWithFuncParams(&err, param1, param2, param3)
```

Where:
- `err` is the actual name of the error result variable
- Parameters are listed by name only, comma-separated

Special cases:
- 0 parameters: `defer errs.WrapWith0FuncParams(&err)`
- 1 parameter: `defer errs.WrapWith1FuncParam(&err, p0)`
- 2 parameters: `defer errs.WrapWith2FuncParams(&err, p0, p1)`
- 3 parameters: `defer errs.WrapWith3FuncParams(&err, p0, p1, p2)`
- 4 parameters: `defer errs.WrapWith4FuncParams(&err, p0, p1, p2, p3)`
- 5 parameters: `defer errs.WrapWith5FuncParams(&err, p0, p1, p2, p3, p4)`
- 6 parameters: `defer errs.WrapWith6FuncParams(&err, p0, p1, p2, p3, p4, p5)`
- 7 parameters: `defer errs.WrapWith7FuncParams(&err, p0, p1, p2, p3, p4, p5, p6)`
- 8 parameters: `defer errs.WrapWith8FuncParams(&err, p0, p1, p2, p3, p4, p5, p6, p7)`
- 9 parameters: `defer errs.WrapWith9FuncParams(&err, p0, p1, p2, p3, p4, p5, p6, p7, p8)`
- 10 parameters: `defer errs.WrapWith10FuncParams(&err, p0, p1, p2, p3, p4, p5, p6, p7, p8, p9)`
- 11+ parameters: `defer errs.WrapWithFuncParams(&err, p0, p1, ...)` (variadic version)
- Receiver methods: Don't include receiver parameter
- Variadic parameters: Use the variadic name without `...`

#### 5. Output Handling

**In-place modification (no -out flag):**
- Modify source files directly (same as go-enum)

**With -out flag:**
- Source is file: Write modified content to -out path
- Source is directory:
  1. Recreate directory structure at -out path
  2. Copy non-Go files unchanged
  3. Write modified Go files

### Algorithm

```
main():
    parse flags and arguments
    validate command (remove or replace)
    validate path exists

    if -out specified:
        setup output handling

    if path is file:
        process_single_file(path)
    else:
        process_directory_recursive(path)

process_file(path):
    use astvisit.RewriteWithReplacements:
        for each AST node:
            if is_target_node(node):
                if command == "remove":
                    add_removal(node)
                else: # replace
                    func_ctx = find_enclosing_function(node)
                    if func_ctx has named error result:
                        replacement = generate_wrap_statement(func_ctx)
                        add_replacement(node, replacement)
                    else:
                        print error, continue

find_enclosing_function(node):
    walk up AST path to find FuncDecl or FuncLit
    return FunctionContext{
        params: extract_param_names(func.Type.Params),
        error_result_name: find_error_result(func.Type.Results),
    }

generate_wrap_statement(ctx):
    if len(params) == 0:
        return "defer errs.WrapWithFuncParams(&" + ctx.error_result_name + ")"
    else:
        params_str = join(ctx.params, ", ")
        return "defer errs.WrapWithFuncParams(&" + ctx.error_result_name + ", " + params_str + ")"
```

### Edge Cases

1. **Anonymous functions (closures):**
   - Handle `*ast.FuncLit` in addition to `*ast.FuncDecl`
   - Use the closure's parameters, not the enclosing function's

2. **Blank identifier as parameter name (`_`):**
   - Ignore as if it was not a parameter

3. **Multiple error results:**
   - Should only have one error result; if multiple, use the last named one of type `error` and output a warning

4. **No named error result:**
   - Print error: `"error: function at <file:line> has no named error result, skipping"`
   - Continue processing other files/functions

5. **Generic functions:**
   - Parameter names work the same; type parameters are separate

6. **Embedded //#wrap-result-err in multi-line comments:**
   - Only match in single-line comments (`//`)

7. **Multiple wrap statements in same function:**
   - Process each independently (though typically there should only be one)

8. **Existing correct wrap statement:**
   - Replace anyway (might have outdated parameter list)

### Import Handling

When replacing/adding wrap statements:
- Ensure `"github.com/domonda/go-errs"` is imported
- Use `astvisit.Imports` to add required imports

### Testing Considerations

Create test files with various scenarios:
- Simple function with parameters
- Function with no parameters
- Function with no named error result (should error)
- Anonymous function/closure
- Nested functions
- Method with receiver
- Existing defer errs.Wrap statement
- //#wrap-result-err marker comment
- Mix of valid and invalid functions in same file

## Example Transformations

### Input 1: Existing wrap statement
```go
func ProcessData(ctx context.Context, id string, count int) (result string, err error) {
    defer errs.WrapWithFuncParams(&err, ctx, id) // outdated params
    // ...
}
```

### Output 1: Updated wrap statement
```go
func ProcessData(ctx context.Context, id string, count int) (result string, err error) {
    defer errs.WrapWithFuncParams(&err, ctx, id, count)
    // ...
}
```

### Input 2: Marker comment
```go
func LoadFile(path string) (data []byte, err error) {
    //#wrap-result-err
    // ...
}
```

### Output 2: Generated wrap statement
```go
func LoadFile(path string) (data []byte, err error) {
    defer errs.WrapWithFuncParams(&err, path)
    // ...
}
```

### Input 3: Remove command (defer statement)
```go
func DoSomething(x int) error {
    defer errs.WrapWithFuncParams(&err, x)
    // ...
}
```

### Output 3: Removed
```go
func DoSomething(x int) error {
    // ...
}
```

### Input 4: Remove command (marker comment)
```go
func LoadData(id string) (data []byte, err error) {
    //#wrap-result-err
    // ...
}
```

### Output 4: Marker removed
```go
func LoadData(id string) (data []byte, err error) {
    // ...
}
```

## Implementation Steps

1. **Setup main.go structure** - CLI parsing, help text
2. **Implement file discovery** - Recursive directory walking, test file exclusion
3. **Implement remove command** - Find and remove defer errs.Wrap statements
4. **Implement replace command** - Find targets, extract function context, generate replacements
5. **Implement -out flag handling** - Copy directory structure, handle non-Go files
6. **Add error handling** - Named error result validation, file I/O errors
7. **Testing** - Manual testing with sample files
8. **Examples** - Add an `examples` subdirectory with sample .go files for the various scenarios with expected output files having the same name but with `.expected.go` suffix. Add a go test that runs the tool on the examples and compares the output name with `.output.go` to the expected output and deletes processed files after comparing with the expected output files. Don't delete the expected output files.
