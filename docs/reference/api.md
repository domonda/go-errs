# Package API Reference

Complete reference for the exported surface of `github.com/domonda/go-errs`.
Import the package as `errs`:

```go
import "github.com/domonda/go-errs"
```

Every function that captures a call stack records the program counters at the
point it runs, so wrap at the site where the error is produced. Formatting is
lazy: the call stack is only rendered into text when `Error()` is called.

For tunable package variables (`MaxCallStackFrames`, `Printer`,
`FormatParamMaxLen`, `TrimFilePathPrefix`, `FormatFunctionCall`) see
[configuration.md](configuration.md). For the command-line tool see
[go-errs-wrap.md](go-errs-wrap.md).

## Contents

- [Error creation](#error-creation)
- [Wrapping with a call stack](#wrapping-with-a-call-stack)
- [Wrapping with function parameters](#wrapping-with-function-parameters)
- [Not-found errors](#not-found-errors)
- [Context errors](#context-errors)
- [Panic recovery](#panic-recovery)
- [Logging control](#logging-control)
- [Secrets](#secrets)
- [Unwrapping and inspection](#unwrapping-and-inspection)
- [Iterators](#iterators)
- [Sentry interop](#sentry-interop)

---

## Error creation

### `func New(text string) error`

Returns a new error whose message is `text`, wrapped with the current call
stack. The underlying error is a [`Sentinel`](#type-sentinel), so
`errors.Is`/`errors.As` still see the plain string error underneath the stack
wrapper.

```go
func DoSomething() error {
    return errs.New("something went wrong")
}
```

Use `errs.New` instead of `errors.New` to get a call stack for free.

### `func Errorf(format string, a ...any) error`

Wraps the result of `fmt.Errorf(format, a...)` with the current call stack.
Supports the `%w` verb for wrapping: the returned error implements `Unwrap` and
participates in `errors.Is`/`errors.As`. Passing more than one `%w`, or a `%w`
operand that is not an `error`, is invalid (same rules as `fmt.Errorf`).

```go
return errs.Errorf("failed to read config %s: %w", path, err)
```

Use `errs.Errorf` instead of `fmt.Errorf`.

### `type Sentinel string`

A string type that implements `error`. Meant for declaring `const` sentinel
errors, which plain `errors.New` cannot because it returns a pointer.

```go
const ErrUserNotFound errs.Sentinel = "user not found"
```

```go
func (s Sentinel) Error() string
```

### `const ErrNotFound Sentinel = "not found"`

A universal "not found" sentinel. Return it directly when a function looks up
exactly one kind of resource, or wrap it to build a specific error. See
[Not-found errors](#not-found-errors) for the matching helpers, and prefer
[`IsErrNotFound`](#func-iserrnotfounderr-error-bool) over
`errors.Is(err, ErrNotFound)` so `sql.ErrNoRows` and `os.ErrNotExist` are also
caught.

---

## Wrapping with a call stack

### `func WrapWithCallStack(err error) error`

Wraps `err` with the current call stack. Returns `nil` if `err` is `nil`.

### `func WrapWithCallStackSkip(skip int, err error) error`

Same as `WrapWithCallStack` but skips `skip` stack frames before capturing.
Use `skip=0` to capture from the immediate caller; add 1 for each extra wrapper
function you route the call through. Returns `nil` if `err` is `nil`.

```go
// Helper that wraps: skip=1 so the caller of the helper appears in the stack
func wrapDatabaseError(err error) error {
    return errs.WrapWithCallStackSkip(1, fmt.Errorf("database error: %w", err))
}
```

Only the top frame captured by each wrapper is printed, so wrapping the same
error twice adds two lines to the rendered stack. See
[call-stacks-and-wrapper-types.md](../explanation/call-stacks-and-wrapper-types.md)
for how skip counts and wrapper reuse interact.

---

## Wrapping with function parameters

These functions take a pointer to a named `error` result and, when that result
is non-nil, replace it with a wrapper that records the call stack **and** the
function's parameter values. They are no-ops when `*resultVar == nil`, which is
why they are safe to `defer` unconditionally.

### `func WrapWithFuncParams(resultVar *error, params ...any)`

The most common entry point. Call it in a `defer` on the first line of a
function with a named `error` result:

```go
func ProcessUser(userID string, age int) (err error) {
    defer errs.WrapWithFuncParams(&err, userID, age)

    if age < 0 {
        return errors.New("invalid age")
    }
    return database.UpdateUser(userID, age)
}
```

When an error is returned the message renders as:

```
invalid age
main.ProcessUser(`user-123`, -5)
    /path/to/file.go:45
```

For a function with 0-10 parameters, prefer the numbered variants below; they
avoid the variadic slice allocation.

### `func WrapWithFuncParamsSkip(skip int, resultVar *error, params ...any)`

Same as `WrapWithFuncParams` but skips `skip` extra stack frames, for building
your own wrapper helpers. `skip=0` is the correct value for a direct `defer`.

### `func WrapWith0FuncParams(resultVar *error)` … `func WrapWith10FuncParams(resultVar *error, p0, …, p9 any)`

Allocation-optimized variants for a fixed parameter count. Note the singular
name for one parameter:

| Function                         | Parameters                       |
| -------------------------------- | -------------------------------- |
| `WrapWith0FuncParams(&err)`      | none                             |
| `WrapWith1FuncParam(&err, p0)`   | 1 (note: `FuncParam`, singular)  |
| `WrapWith2FuncParams(&err, …)`   | 2                                |
| `WrapWith3FuncParams(&err, …)`   | 3                                |
| `WrapWith4FuncParams(&err, …)`   | 4                                |
| `WrapWith5FuncParams(&err, …)`   | 5                                |
| `WrapWith6FuncParams(&err, …)`   | 6                                |
| `WrapWith7FuncParams(&err, …)`   | 7                                |
| `WrapWith8FuncParams(&err, …)`   | 8                                |
| `WrapWith9FuncParams(&err, …)`   | 9                                |
| `WrapWith10FuncParams(&err, …)`  | 10                               |

For 11+ parameters, use the variadic `WrapWithFuncParams`. The
[`go-errs-wrap`](go-errs-wrap.md) tool picks the right variant for you.

---

## Not-found errors

### `func IsErrNotFound(err error) bool`

Reports whether `err` is non-nil and unwraps to any of
[`ErrNotFound`](#const-errnotfound-sentinel--not-found), `sql.ErrNoRows`, or
`os.ErrNotExist`.

### `func IsOtherThanErrNotFound(err error) bool`

Reports whether `err` is non-nil and is **not** any of those three. Useful for
"a real error occurred" checks where a missing resource is expected.

### `func ReplaceErrNotFound(err, replacement error) error`

Returns `replacement` if `IsErrNotFound(err)` is true, otherwise returns `err`
unchanged. Lets you swap any not-found variant for a domain-specific error.

---

## Context errors

All four consult the context's `Done` channel and error without blocking.

### `func IsContextCanceled(ctx context.Context) bool`

True if `ctx` is done and its error unwraps to `context.Canceled`.

### `func IsContextDeadlineExceeded(ctx context.Context) bool`

True if `ctx` is done and its error unwraps to `context.DeadlineExceeded`.

### `func IsContextDone(ctx context.Context) bool`

True if `ctx`'s `Done` channel is closed (canceled or deadline exceeded).

### `func IsContextError(err error) bool`

True if `err` is non-nil and unwraps to `context.Canceled` or
`context.DeadlineExceeded`. Use it to avoid retrying on context errors.

---

## Panic recovery

### `func AsError(val any) error`

Converts any value to an `error` without wrapping. `nil` → `nil`; an `error` is
returned as-is; `[]error` is joined via `errors.Join`; `string` and
`fmt.Stringer` become their text; anything else is `fmt.Sprint`-ed.

### `func AsErrorWithDebugStack(val any) error`

Like `AsError`, but non-nil results are wrapped with `debug.Stack()` after a
newline. This is the goroutine stack at recovery time (broader than the
call-stack wrappers), useful for panic diagnostics.

### `func RecoverPanicAsError(result *error)`

`defer` this to recover a panic and store it in `*result`. If `*result` is
already non-nil, the panic wraps the existing error with a
`function returning error (…) panicked with: …` message.

```go
func RiskyOperation() (err error) {
    defer errs.RecoverPanicAsError(&err)
    return doSomethingRisky() // may panic
}
```

### `func RecoverPanicAsErrorWithFuncParams(result *error, params ...any)`

Like `RecoverPanicAsError`, but also records the function's parameters into the
recovered error's call stack.

### `func LogPanicWithFuncParams(log Logger, params ...any)`

Recovers a panic, builds an error with the debug stack and parameters, logs it
to `log` with prefix `LogPanicWithFuncParams: `, then **re-panics**. Use it to
add observability without swallowing the panic.

### `func RecoverAndLogPanicWithFuncParams(log Logger, params ...any)`

Same as `LogPanicWithFuncParams` but **does not re-panic** — it recovers, logs
with prefix `RecoverAndLogPanicWithFuncParams: `, and lets the function return
normally.

---

## Logging control

### `type Logger interface`

```go
type Logger interface {
    Printf(format string, args ...any)
}
```

The minimal logging sink used by the panic-logging and `LogFunctionCall`
helpers. `*log.Logger` and most structured loggers satisfy it.

### `type LogDecisionMaker interface`

```go
type LogDecisionMaker interface {
    error
    ShouldLog() bool
}
```

Implement this on an error type to control whether callers log it.

### `func ShouldLog(err error) bool`

Returns `false` for a `nil` error. Otherwise, if `err` unwraps to a
`LogDecisionMaker`, returns that error's `ShouldLog()`; if not, returns `true`.

```go
if errs.ShouldLog(err) {
    logger.Error(err)
}
```

### `func DontLog(err error) error`

Wraps `err` so that `ShouldLog` returns `false` for it. Returns `nil` for a
`nil` error. The wrapper still unwraps to the original error.

### `func LogFunctionCall(logger Logger, function string, params ...any)`

If `logger` is non-nil, formats `function` and `params` with
[`FormatFunctionCall`](configuration.md#formatfunctioncall) and logs the result.
A no-op when `logger` is `nil`.

---

## Secrets

### `type Secret interface`

```go
type Secret interface {
    Secret() any    // the wrapped value
    String() string // a redacted placeholder, never the real value
}
```

A value wrapper that prevents its contents from appearing in logs or error call
stacks. It also implements `pretty.Stringer` (via `PrettyString()`), so the
`go-pretty` formatter used for parameters redacts it recursively, even when
nested inside a struct.

### `func KeepSecret(val any) Secret`

Wraps `val` in a `Secret`. `String()` and `PrettyString()` return
`***REDACTED***`; `Secret()` returns the original value.

```go
func Login(username, password string) (err error) {
    defer errs.WrapWithFuncParams(&err, username, errs.KeepSecret(password))
    // renders: Login(`admin`, ***REDACTED***)
    return authenticate(username, password)
}
```

Prefer `KeepSecret(param)` over dropping a sensitive parameter from the wrap
call: [`go-errs-wrap replace`](go-errs-wrap.md) preserves `KeepSecret` wrappers
but re-adds omitted parameters. See
[redact-sensitive-parameters.md](../how-to/redact-sensitive-parameters.md).

---

## Unwrapping and inspection

### `func Root(err error) error`

Recursively unwraps `err` to its root cause using both `Unwrap() error` and
`Unwrap() []error`. For a multi-error tree (`errors.Join`), it returns the root
of the first non-nil branch.

### `func UnwrapCallStack(err error) error`

Removes only the **top-level** call-stack wrappers (including those carrying
function parameters) and returns the underlying error, preserving the rest of
the chain. Unlike `Root`, it stops at the first non-stack wrapper.

```go
err := errs.WrapWithCallStack(fmt.Errorf("context: %w", sentinel))
clean := errs.UnwrapCallStack(err) // == fmt.Errorf("context: %w", sentinel)
```

Handy for comparing errors without call-stack noise (two wraps of the same
sentinel are unequal, but their `UnwrapCallStack` results are equal).

### `func Has[T error](err error) bool`

Reports whether `err`'s tree contains an error of type `T`. A shortcut for
`errors.As` when you do not need the matched value.

```go
if errs.Has[*DatabaseError](err) { /* … */ }
```

Since Go 1.26 the standard library offers `errors.AsType[T](err) (T, bool)`;
use it when you need the value, `Has` when you only need the boolean.

### `func As[T error](err error) []T`

Returns **all** errors of type `T` in the full wrapping tree, traversing both
`Unwrap() error` and `Unwrap() []error`. Unlike `errors.AsType` (which returns
only the first match), this collects every match across `errors.Join` branches.

```go
err := errors.Join(
    &ValidationError{Field: "name"},
    &ValidationError{Field: "email"},
)
errs.As[*ValidationError](err) // both "name" and "email"
```

An error matches `T` if the type assertion holds, or if it has an
`As(any) bool` method that returns true for a non-nil `*T`.

### `func Type[T error](err error) bool`

Reports whether `err` is non-nil and it, or any unwrapped error, has the concrete
type `T`. Like `errors.As` but without assigning a target and without consulting
`Is`/`As` methods.

### `func IsType(err, ref error) bool`

Reports whether `err`, or any unwrapped error, has the same concrete type as
`ref`. The non-generic counterpart to `Type`.

---

## Iterators

### `func IterSeq(err error) iter.Seq[error]`

Returns an `iter.Seq[error]` (Go 1.23+ range-over-func) that yields `err` once.

```go
for err := range errs.IterSeq(myErr) {
    // process err
}
```

### `func IterSeq2[T any](err error) iter.Seq2[T, error]`

Returns an `iter.Seq2[T, error]` that yields the zero value of `T` and `err`
once. Fits the "value, error" two-value iterator pattern where you need to
surface an error before any value is produced.

---

## Sentry interop

Errors wrapped by this package expose a `StackTrace() []uintptr` method (the
`pkg/errors` shape). `sentry-go`'s `sentry.ExtractStacktrace` discovers it by
reflection, so wrapped errors carry their call stack into Sentry events with no
import of, or dependency on, `sentry-go`:

```go
hub.CaptureException(err) // err's call stack appears in the Sentry event
```

`StackTrace()` is a method on the internal wrapper type, not a package-level
function, so there is no symbol to call directly. See
[send-stack-traces-to-sentry.md](../how-to/send-stack-traces-to-sentry.md) for
usage and
[sentry-stack-trace-interop.md](../explanation/sentry-stack-trace-interop.md)
for why the method exists.

---

## Related

- [configuration.md](configuration.md) — tunable package variables
- [go-errs-wrap.md](go-errs-wrap.md) — the code-transformation CLI
- [How-to guides](../how-to/) — task-oriented recipes
- [Explanation](../explanation/) — design rationale
