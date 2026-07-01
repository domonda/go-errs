# Configuration Reference

`errs` exposes a small set of package-level variables that control how call
stacks and parameters are captured and rendered. Set them once at process start
(for example in an `init` function or early in `main`) before errors are
created; they are read whenever an error is formatted or captured.

All variables live in the `errs` package:

```go
import "github.com/domonda/go-errs"
```

## Contents

- [`TrimFilePathPrefix`](#trimfilepathprefix)
- [`MaxCallStackFrames`](#maxcallstackframes)
- [`FormatParamMaxLen`](#formatparammaxlen)
- [`Printer`](#printer)
- [`FormatFunctionCall`](#formatfunctioncall)

---

## `TrimFilePathPrefix`

```go
var TrimFilePathPrefix string
```

A prefix trimmed from the start of every file path shown in a rendered call
stack. It defaults to the `.../src/` portion of the build environment path
(everything up to `github.com` in this package's own source path), or is empty
when the binary was built with `-trimpath` or lives outside a `github.com`
path.

Set it to shorten stack paths to something repo-relative:

```go
errs.TrimFilePathPrefix = "/go/src/"
```

**Type:** `string`
**Default:** build-environment `src` prefix, or `""`

---

## `MaxCallStackFrames`

```go
var MaxCallStackFrames = 32
```

The maximum number of stack frames captured per wrap. Capture allocates a
`[]uintptr` of this length and fills it via `runtime.Callers`, so larger values
trade memory for deeper stacks.

```go
errs.MaxCallStackFrames = 64
```

**Type:** `int`
**Default:** `32`

---

## `FormatParamMaxLen`

```go
var FormatParamMaxLen = 5000
```

The maximum length, in bytes, of a single formatted parameter value in a
rendered call stack. When a parameter's formatted representation exceeds this,
it is truncated (on a valid-UTF-8 boundary) and suffixed with `…(TRUNCATED)`.
This keeps large strings, JSON blobs, or data structures from making errors
unreadable.

```go
errs.FormatParamMaxLen = 500

func ProcessDocument(content string) (err error) {
    defer errs.WrapWithFuncParams(&err, content)
    // a 2KB content renders as:
    // ProcessDocument("first 500 bytes…(TRUNCATED)")
    return parseDocument(content)
}
```

**Type:** `int`
**Default:** `5000`

---

## `Printer`

```go
var Printer *pretty.Printer
```

The `github.com/domonda/go-pretty` printer used to format each function
parameter. It is created with go-pretty's default string/error/slice length
limits. Parameters are formatted by checking, in priority order:

1. `pretty.PrintableWithResult` — `PrettyPrint(io.Writer) (int, error)`
2. `pretty.Printable` — `PrettyPrint(io.Writer)`
3. `pretty.Stringer` — `PrettyString() string`

This works recursively for nested struct fields, so a type implementing any of
these interfaces is formatted correctly even when embedded in another value.

Replace the printer with `WithPrintFuncFor` to intercept formatting — for
example to redact secrets by type, struct tag, or value pattern:

```go
func init() {
    errs.Printer = errs.Printer.WithPrintFuncFor(func(v reflect.Value) pretty.PrintFunc {
        if v.Kind() == reflect.String && strings.Contains(v.String(), "secret") {
            return func(w io.Writer) (int, error) {
                return io.WriteString(w, "`***REDACTED***`")
            }
        }
        return pretty.PrintFuncForPrintable(v) // fall back to default
    })
}
```

`WithPrintFuncFor` returns a **new** printer with the hook installed. Always
return `pretty.PrintFuncForPrintable(v)` as the fallback so the default
interface checks still apply to values your hook does not handle.

**Type:** `*pretty.Printer`
**Default:** a `pretty.Printer` using `pretty.DefaultPrinter`'s length limits

See [secret-redaction-and-pretty-printing.md](../explanation/secret-redaction-and-pretty-printing.md)
for the full model and
[redact-sensitive-parameters.md](../how-to/redact-sensitive-parameters.md) for
recipes.

---

## `FormatFunctionCall`

```go
var FormatFunctionCall = func(function string, params ...any) string { … }
```

A function variable that renders a function call as pseudo-syntax
(`functionName(param1, param2, …)`) for the parameter lines of a call stack.
The default implementation formats each parameter with [`Printer`](#printer) and
truncates any result longer than [`FormatParamMaxLen`](#formatparammaxlen).

Reassign it to change formatting globally:

```go
errs.FormatFunctionCall = func(function string, params ...any) string {
    return fmt.Sprintf("%s(%v)", function, params)
}
```

`FormatFunctionCall` is also used by
[`LogFunctionCall`](api.md#func-logfunctioncalllogger-logger-function-string-params-any),
which logs a formatted call to a [`Logger`](api.md#type-logger-interface).

**Type:** `func(function string, params ...any) string`
**Default:** go-pretty-based formatter with `FormatParamMaxLen` truncation

---

## Related

- [api.md](api.md) — the full package API
- [Secret redaction](../explanation/secret-redaction-and-pretty-printing.md)
