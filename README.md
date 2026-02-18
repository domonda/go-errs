# go-errs

Go error wrapping with call-stack and function parameter capture.

[![Go Reference](https://pkg.go.dev/badge/github.com/domonda/go-errs.svg)](https://pkg.go.dev/github.com/domonda/go-errs)
[![Go Report Card](https://goreportcard.com/badge/github.com/domonda/go-errs)](https://goreportcard.com/report/github.com/domonda/go-errs)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)


## Features

- **Automatic call stack capture** - Every error wrapped with this package includes the full call stack at the point where the error was created or wrapped
- **Function parameter tracking** - Capture and display function parameters in error messages for detailed debugging
- **Error wrapping compatible** - Works seamlessly with `errors.Is`, `errors.As`, and `errors.Unwrap`
- **Helper utilities** - Common patterns for NotFound errors, context errors, and panic recovery
- **Customizable formatting** - Control how sensitive data appears in error messages
- **Iterator support** - Convert errors to `iter.Seq` and `iter.Seq2` iterators

## Installation

```bash
go get github.com/domonda/go-errs
```

## Quick Start

### Basic Error Creation

```go
import "github.com/domonda/go-errs"

func DoSomething() error {
    return errs.New("something went wrong")
}

// Error output includes call stack:
// something went wrong
// main.DoSomething
//     /path/to/file.go:123
```

### Wrapping Errors with Function Parameters

The most powerful feature - automatically capture function parameters when errors occur:

```go
func ProcessUser(userID string, age int) (err error) {
    defer errs.WrapWithFuncParams(&err, userID, age)

    if age < 0 {
        return errors.New("invalid age")
    }

    return database.UpdateUser(userID, age)
}

// When an error occurs, output includes:
// invalid age
// main.ProcessUser("user-123", -5)
//     /path/to/file.go:45
```

### Error Wrapping

```go
func LoadConfig(path string) error {
    data, err := os.ReadFile(path)
    if err != nil {
        return errs.Errorf("failed to read config: %w", err)
    }
    // ... parse data
    return nil
}
```

## Core Functions

### Error Creation

- **`errs.New(text)`** - Create a new error with call stack
- **`errs.Errorf(format, ...args)`** - Format an error with call stack (supports `%w` for wrapping)
- **`errs.Sentinel(text)`** - Create a const-able sentinel error

### Error Wrapping with Parameters

- **`errs.WrapWithFuncParams(&err, params...)`** - Most common: wrap error with function parameters
- **`errs.WrapWith0FuncParams(&err)`** through **`errs.WrapWith10FuncParams(&err, p0, ...)`** - Optimized variants for specific parameter counts
- **`errs.WrapWithFuncParamsSkip(skip, &err, params...)`** - Advanced: control stack frame skipping

### Error Wrapping without Parameters

- **`errs.WrapWithCallStack(err)`** - Wrap error with call stack only
- **`errs.WrapWithCallStackSkip(skip, err)`** - Advanced: control stack frame skipping

## Advanced Features

### NotFound Errors

Standardized "not found" error handling compatible with `sql.ErrNoRows` and `os.ErrNotExist`:

```go
var ErrUserNotFound = fmt.Errorf("user %w", errs.ErrNotFound)

func GetUser(id string) (*User, error) {
    user, err := db.Query("SELECT * FROM users WHERE id = ?", id)
    if errs.IsErrNotFound(err) {
        return nil, ErrUserNotFound
    }
    return user, err
}

// Check for any "not found" variant
if errs.IsErrNotFound(err) {
    // Handle not found case
}
```

### Context Error Helpers

```go
// Check if context is done
if errs.IsContextDone(ctx) {
    // Handle context done
}

// Check specific context errors
if errs.IsContextCanceled(ctx) {
    // Handle cancellation
}

if errs.IsContextDeadlineExceeded(ctx) {
    // Handle timeout
}

// Check if an error is context-related
if errs.IsContextError(err) {
    // Don't retry context errors
}
```

### Panic Recovery

```go
func RiskyOperation() (err error) {
    defer errs.RecoverPanicAsError(&err)

    // If this panics, it will be converted to an error
    return doSomethingRisky()
}

// With function parameters
func ProcessItem(id string) (err error) {
    defer errs.RecoverPanicAsErrorWithFuncParams(&err, id)

    return processItem(id) // May panic
}
```

### Logging Control

```go
type CustomError struct {
    error
}

func (e CustomError) ShouldLog() bool {
    return false // Don't log this error
}

// Check if error should be logged
if errs.ShouldLog(err) {
    logger.Error(err)
}

// Wrap error to prevent logging
err = errs.DontLog(err)
```

### Protecting Sensitive Data

#### Using KeepSecret for Quick Protection

For simple cases where you want to prevent a parameter from appearing in logs, use `errs.KeepSecret(param)`:

```go
func Login(username string, password string) (err error) {
    defer errs.WrapWithFuncParams(&err, username, errs.KeepSecret(password))
    // Error messages will show: Login("admin", ***REDACTED***)
    return authenticate(username, password)
}
```

The `Secret` interface wraps a value and ensures it's never logged or printed:
- `String()` returns `"***REDACTED***"`
- Implements `pretty.Stringer` (via `PrettyString()`) to ensure redaction in error call stacks and pretty-printed output
- Use `secret.Secret()` to retrieve the actual value when needed

**Important:** Wrapping a parameter with `errs.KeepSecret()` is preferable to omitting it entirely from a `defer errs.WrapWith*` statement. When you run `go-errs-wrap replace`, omitted parameters will be added back, but `KeepSecret`-wrapped parameters are preserved in their wrapped form.

#### Custom Types with go-pretty Interfaces

For custom types, implement one of the [go-pretty](https://github.com/domonda/go-pretty) interfaces to control how they appear in error messages. The interfaces are checked in priority order:

1. **`pretty.PrintableWithResult`** — `PrettyPrint(io.Writer) (n int, err error)` — full control with byte count
2. **`pretty.Printable`** — `PrettyPrint(io.Writer)` — simple writer-based formatting
3. **`pretty.Stringer`** — `PrettyString() string` — simplest, just return a string

```go
// Simplest approach: implement pretty.Stringer
type Password string

func (Password) PrettyString() string {
    return "***REDACTED***"
}

func Login(username string, pwd Password) (err error) {
    defer errs.WrapWithFuncParams(&err, username, pwd)
    // Error messages will show: Login("admin", ***REDACTED***)
    return authenticate(username, pwd)
}
```

The `go-pretty` library automatically handles recursive formatting, so types implementing any of these interfaces
will be properly formatted even when nested in other structs.

#### Customizing Call Stack Printing with PrintFuncFor

**Default behavior:** By default, the `Printer` checks if a type implements `pretty.PrintableWithResult`, `pretty.Printable`, or `pretty.Stringer` (in that order). This is the recommended approach for types you own and control.

For advanced use cases where you need to customize formatting beyond implementing these interfaces, use `Printer.WithPrintFuncFor()`. This is useful when:
- You want to format types you don't control (third-party types, stdlib types)
- You need runtime-conditional formatting based on values or context
- You want to adapt types that implement other interfaces (e.g., `fmt.Stringer`)
- You need to apply global formatting rules based on struct tags or type patterns
- You want to override or wrap the default behavior

```go
import (
    "fmt"
    "io"
    "reflect"
    "strings"
    "github.com/domonda/go-errs"
    "github.com/domonda/go-pretty"
)

func init() {
    // Create a custom printer with PrintFuncFor hook
    errs.Printer = errs.Printer.WithPrintFuncFor(func(v reflect.Value) pretty.PrintFunc {
        // Example 1: Mask strings containing sensitive keywords
        if v.Kind() == reflect.String {
            str := v.String()
            if strings.Contains(strings.ToLower(str), "password") ||
               strings.Contains(strings.ToLower(str), "token") ||
               strings.Contains(strings.ToLower(str), "apikey") {
                return func(w io.Writer) (int, error) {
                    return io.WriteString(w, "`***REDACTED***`")
                }
            }
        }

        // Example 2: Hide struct fields tagged with `secret:"true"`
        if v.Kind() == reflect.Struct {
            t := v.Type()
            for i := range t.NumField() {
                field := t.Field(i)
                if field.Tag.Get("secret") == "true" {
                    // Create a custom formatter that masks tagged fields
                    return func(w io.Writer) (int, error) {
                        // Custom struct formatting logic here
                        n1, _ := io.WriteString(w, t.Name())
                        n2, err := io.WriteString(w, "{***FIELDS_REDACTED***}")
                        return n1 + n2, err
                    }
                }
            }
        }

        // Example 3: Adapt types implementing fmt.Stringer
        stringer, ok := v.Interface().(fmt.Stringer)
        if !ok && v.CanAddr() {
            stringer, ok = v.Addr().Interface().(fmt.Stringer)
        }
        if ok {
            return func(w io.Writer) (int, error) {
                return io.WriteString(w, stringer.String())
            }
        }

        // Fall back to default Printable interface handling
        return pretty.PrintFuncForPrintable(v)
    })
}

// Now all error call stacks will use your custom formatting
func ProcessPayment(amount int, cardNumber string) (err error) {
    defer errs.WrapWithFuncParams(&err, amount, cardNumber)
    // If cardNumber contains "4111-1111-1111-1111", it will be shown as:
    // ProcessPayment(1000, `***REDACTED***`)
    return chargeCard(amount, cardNumber)
}
```

**Key points:**
- **Default behavior**: If no `PrintFuncFor` is set, the `Printer` automatically checks if types implement `pretty.PrintableWithResult`, `pretty.Printable`, or `pretty.Stringer`
- `WithPrintFuncFor()` returns a new `Printer` with the custom hook installed
- The hook receives a `reflect.Value` and returns a `pretty.PrintFunc` (or `nil` to use default)
- **Important**: In a PrintFuncFor function, always return `pretty.PrintFuncForPrintable(v)` as the fallback to preserve the default interface checking behavior.
- The hook applies to all values printed in error call stacks, including nested struct fields
- This approach allows centralized control over sensitive data redaction without modifying type definitions
- `PrintFuncFor` is evaluated for every value, allowing you to intercept and customize formatting before the default interfaces are checked

**Comparison: Interfaces vs PrintFuncFor**

```go
// Approach 1: Implement pretty.Stringer (recommended for types you own)
type APIKey string

func (APIKey) PrettyString() string {
    return "***REDACTED***"
}

// Approach 2: Use PrintFuncFor (for types you don't control or global rules)
func init() {
    errs.Printer = errs.Printer.WithPrintFuncFor(func(v reflect.Value) pretty.PrintFunc {
        // Check for APIKey type from a third-party package
        if v.Type().String() == "thirdparty.APIKey" {
            return func(w io.Writer) (int, error) {
                return io.WriteString(w, "***REDACTED***")
            }
        }
        // Fall back to checking for default interfaces
        return pretty.PrintFuncForPrintable(v)
    })
}
```

**Example with struct tags:**

```go
type User struct {
    ID       string
    Email    string
    Password string `secret:"true"`
    APIKey   string `secret:"true"`
}

// With the PrintFuncFor hook configured above,
// error messages will automatically redact fields tagged with secret:"true"
```

## Unwrapping and Inspection

### Finding Errors by Type

Since Go 1.26, the standard library provides `errors.AsType[E](err) (E, bool)` which returns the first matching error and a boolean. Use it when you need the matched value:

```go
// Go 1.26+: get first matching error (preferred when you need the value)
if dbErr, ok := errors.AsType[*DatabaseError](err); ok {
    log.Println("database error:", dbErr.Code)
}
```

This package provides additional generic helpers:

```go
// Check if error chain contains a specific type (bool only, no value needed)
// Equivalent to: _, ok := errors.AsType[*DatabaseError](err)
if errs.Has[*DatabaseError](err) {
    // Handle database error
}

// Get ALL errors of a specific type from the full wrapping tree.
// Unlike errors.AsType which returns only the first match,
// errs.As traverses the entire tree including multi-errors (errors.Join):
//
//   err := errors.Join(
//       &ValidationError{Field: "name"},
//       &ValidationError{Field: "email"},
//   )
//   errors.AsType[*ValidationError](err) // returns only "name"
//   errs.As[*ValidationError](err)       // returns both "name" and "email"
//
allErrors := errs.As[*ValidationError](err)
for _, vErr := range allErrors {
    log.Println("invalid field:", vErr.Field)
}

// Check error type without custom Is/As methods
if errs.Type[*DatabaseError](err) {
    // Error is or wraps a DatabaseError
}
```

### Navigating Error Chains

```go
// Get the root cause error
rootErr := errs.Root(err)

// Unwrap call stack information only
plainErr := errs.UnwrapCallStack(err)
```

### Go 1.23+ Iterator Support

```go
// Convert error to single-value iterator
for err := range errs.IterSeq(myErr) {
    // Process error
}

// Convert error to two-value iterator (value, error) pattern
for val, err := range errs.IterSeq2[MyType](myErr) {
    if err != nil {
        // Handle error
    }
}
```

## Configuration

### Customize Call Stack Display

```go
// Change path prefix trimming
errs.TrimFilePathPrefix = "/go/src/"

// Adjust maximum stack depth
errs.MaxCallStackFrames = 64 // Default is 32
```

### Limit Parameter Value Length

Control how long parameter values can be in error messages to prevent huge values from making errors unreadable:

```go
// Default is 5000 bytes
errs.FormatParamMaxLen = 500

// Now long parameter values will be truncated
func ProcessDocument(content string) (err error) {
    defer errs.WrapWithFuncParams(&err, content)
    // If content is 2KB, error will show:
    // ProcessDocument("first 500 bytes…(TRUNCATED)")
    return parseDocument(content)
}
```

This is particularly useful when dealing with:
- Large JSON or XML payloads
- Binary data encoded as strings
- Long database query results
- Large data structures

### Customize Function Call Formatting

```go
// Replace the global formatter
errs.FormatFunctionCall = func(function string, params ...any) string {
    // Your custom formatting logic
    return fmt.Sprintf("%s(%v)", function, params)
}
```

## Best Practices

### 1. Always use defer for WrapWithFuncParams

```go
func MyFunc(id string) (err error) {
    defer errs.WrapWithFuncParams(&err, id)
    // Function body
}
```

### 2. Use optimized variants for better performance

```go
// Instead of:
defer errs.WrapWithFuncParams(&err, p0, p1, p2)

// Use:
defer errs.WrapWith3FuncParams(&err, p0, p1, p2)
```

### 3. Protect sensitive data

```go
// Simplest: implement pretty.Stringer
type APIKey string

func (APIKey) PrettyString() string {
    return "***REDACTED***"
}

// Or use errs.KeepSecret for function parameters
defer errs.WrapWithFuncParams(&err, errs.KeepSecret(apiKey))
```

### 4. Use errs.Errorf for wrapping

```go
// Good - preserves error chain
return errs.Errorf("failed to process user %s: %w", userID, err)

// Avoid - loses error chain
return errs.New(fmt.Sprintf("failed: %s", err))
```

### 5. Prefer errs package for all error creation

```go
// Use errs.New instead of errors.New
return errs.New("something failed")

// Use errs.Errorf instead of fmt.Errorf
return errs.Errorf("failed: %w", err)
```

## go-errs-wrap CLI Tool

The `go-errs-wrap` command-line tool helps manage `defer errs.WrapWithFuncParams` statements in your Go code.

### Installation

```bash
go install github.com/domonda/go-errs/cmd/go-errs-wrap@latest
```

### Commands

| Command | Description |
|---------|-------------|
| `remove` | Remove all `defer errs.Wrap*` or `//#wrap-result-err` lines |
| `replace` | Replace existing `defer errs.Wrap*` or `//#wrap-result-err` with properly generated code |
| `insert` | Insert `defer errs.Wrap*` at the first line of functions with named error results that don't already have one |

### Usage Examples

**Insert wrap statements into all functions missing them:**

```bash
# Process a single file
go-errs-wrap insert ./pkg/mypackage/file.go

# Process all Go files in a directory recursively
go-errs-wrap insert ./pkg/...
```

**Replace outdated wrap statements with correct ones:**

```bash
# Update parameters in existing wrap statements
go-errs-wrap replace ./pkg/mypackage/file.go
```

**Remove all wrap statements:**

```bash
go-errs-wrap remove ./pkg/...
```

**Write changes to another output location:**

```bash
# Output to a different location
go-errs-wrap insert -out ./output ./pkg/mypackage

# Show verbose progress
go-errs-wrap insert -verbose ./pkg/...
```

### Options

| Option | Description |
|--------|-------------|
| `-out <path>` | Output to different location instead of modifying source |
| `-minvariadic` | Use specialized `WrapWithNFuncParams` functions instead of variadic |
| `-validate` | Dry run mode: check for issues without modifying files (useful for CI) |
| `-verbose` | Print progress information |
| `-help` | Show help message |

### Example Transformation

Given this input file:

```go
package example

func ProcessData(ctx context.Context, id string) (err error) {
    return doWork(ctx, id)
}
```

Running `go-errs-wrap insert example.go` produces:

```go
package example

import "github.com/domonda/go-errs"

func ProcessData(ctx context.Context, id string) (err error) {
    defer errs.WrapWith2FuncParams(&err, ctx, id)

    return doWork(ctx, id)
}
```

The tool:
- Inserts the `defer errs.Wrap*` statement at the first line of the function body
- Adds an empty line after the defer statement
- Automatically adds the required import
- Uses the optimized function variant based on parameter count
- Skips functions without named error results
- Skips functions that already have a `defer errs.Wrap*` statement
- Preserves `errs.KeepSecret(param)` wrapped parameters during replacement

### Preserving Secrets During Replacement

When using `go-errs-wrap replace`, parameters wrapped with `errs.KeepSecret()` are preserved. This allows developers to mark sensitive parameters once and have that protection maintained across replacements:

```go
// Before: developer manually wrapped password with KeepSecret
func Login(username, password string) (err error) {
    defer errs.WrapWithFuncParams(&err, username, errs.KeepSecret(password))
    return authenticate(username, password)
}

// After running: go-errs-wrap replace -minvariadic file.go
// The KeepSecret wrapping is preserved:
func Login(username, password string) (err error) {
    defer errs.WrapWith2FuncParams(&err, username, errs.KeepSecret(password))
    return authenticate(username, password)
}
```

This is preferable to omitting sensitive parameters entirely, as omitted parameters would be re-added by the tool.

## Compatibility

- **Go version:** Requires Go 1.24+
- **Error handling:** Fully compatible with `errors.Is`, `errors.As`, `errors.Unwrap`, and `errors.Join`
- **Testing:** Use with `testify` or any testing framework

## Performance

- Zero-allocation error wrapping for functions with 0-10 parameters (using specialized functions)
- Efficient call stack capture using `runtime.Callers`
- Lazy error message formatting - only formats when `Error()` is called
- Configurable stack depth to balance detail vs memory usage

## Examples

See the [godoc](https://pkg.go.dev/github.com/domonda/go-errs) for more examples.

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Contributing

Contributions welcome! Please open an issue or submit a pull request.

## Related Packages

- [go-pretty](https://github.com/domonda/go-pretty) - Pretty printing used for error parameter formatting
