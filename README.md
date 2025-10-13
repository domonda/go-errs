# go-errs

Go 1.13+ compatible error wrapping with call stacks and function parameters.

[![Go Reference](https://pkg.go.dev/badge/github.com/domonda/go-errs.svg)](https://pkg.go.dev/github.com/domonda/go-errs)
[![Go Report Card](https://goreportcard.com/badge/github.com/domonda/go-errs)](https://goreportcard.com/report/github.com/domonda/go-errs)

## Features

- **Automatic call stack capture** - Every error wrapped with this package includes the full call stack at the point where the error was created or wrapped
- **Function parameter tracking** - Capture and display function parameters in error messages for detailed debugging
- **Go 1.13+ error wrapping compatible** - Works seamlessly with `errors.Is`, `errors.As`, and `errors.Unwrap`
- **Zero allocation optimization** - Specialized functions for 0-10 parameters to avoid varargs allocations
- **Helper utilities** - Common patterns for NotFound errors, context errors, and panic recovery
- **Customizable formatting** - Control how sensitive data appears in error messages
- **Go 1.23+ iterator support** - Convert errors to iterators for functional programming patterns

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

### Customizing Sensitive Data Display

Implement `CallStackPrintable` to control how your types appear in error messages:

```go
type Password struct {
    value string
}

func (p Password) PrintForCallStack(w io.Writer) {
    w.Write([]byte("***REDACTED***"))
}

func Login(username string, pwd Password) (err error) {
    defer errs.WrapWithFuncParams(&err, username, pwd)
    // Error messages will show: Login("admin", ***REDACTED***)
    return authenticate(username, pwd)
}
```

## Unwrapping and Inspection

### Finding Errors by Type

```go
// Check if error chain contains a specific type
if errs.Has[*DatabaseError](err) {
    // Handle database error
}

// Get all errors of a specific type from the chain
dbErrors := errs.As[*DatabaseError](err)
for _, dbErr := range dbErrors {
    // Handle each database error
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
type APIKey string

func (k APIKey) PrintForCallStack(w io.Writer) {
    io.WriteString(w, "***")
}
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

## Compatibility

- **Go version:** Requires Go 1.13+ for error wrapping, Go 1.23+ for iterator support
- **Error handling:** Fully compatible with `errors.Is`, `errors.As`, `errors.Unwrap`, and `errors.Join`
- **Testing:** Use with `testify` or any testing framework

## Performance

- Zero-allocation error wrapping for functions with 0-10 parameters (using specialized functions)
- Efficient call stack capture using `runtime.Callers`
- Lazy error message formatting - only formats when `Error()` is called
- Configurable stack depth to balance detail vs memory usage

## Examples

See the [examples directory](examples/) and [godoc](https://pkg.go.dev/github.com/domonda/go-errs) for more examples.

## License

MIT License - see [LICENSE](LICENSE) file for details.

## Contributing

Contributions welcome! Please open an issue or submit a pull request.

## Related Packages

- [go-pretty](https://github.com/domonda/go-pretty) - Pretty printing used for error parameter formatting
