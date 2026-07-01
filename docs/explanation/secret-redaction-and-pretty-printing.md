# Secret Redaction and Pretty-Printing

This explains how `errs` keeps sensitive values out of error call stacks, the
priority order it uses to format parameters, and why redaction survives nesting.
For recipes see
[redact-sensitive-parameters.md](../how-to/redact-sensitive-parameters.md).

## The problem

The feature that makes this library useful — recording function parameters in
error messages — is also a liability. A `defer errs.WrapWithFuncParams(&err, …)`
in a `Login` or `ProcessPayment` function will happily print the password, API
key, or card number straight into your logs and error tracker. You need a way
to record *that a parameter was present* without recording *its value*, and it
has to hold even when the secret is buried inside a struct you pass by value.

## The approach

Parameters are not formatted with `fmt`. They go through a
`github.com/domonda/go-pretty` [`Printer`](../reference/configuration.md#printer),
which checks each value against three interfaces in priority order and uses the
first that matches:

```
1. pretty.PrintableWithResult   PrettyPrint(io.Writer) (n int, err error)
2. pretty.Printable             PrettyPrint(io.Writer)
3. pretty.Stringer              PrettyString() string
```

A type controls its own rendering by implementing any one of these. Redaction is
just "render as a placeholder instead of the real value." There are three ways
to reach that, from most local to most global:

### 1. `KeepSecret` at the call site

`errs.KeepSecret(val)` wraps a single value in a `Secret`. The wrapper's
`String()` and `PrettyString()` both return `***REDACTED***`, and `Secret()`
hands back the original value when you genuinely need it:

```go
defer errs.WrapWithFuncParams(&err, username, errs.KeepSecret(password))
// renders: Login(`admin`, ***REDACTED***)
```

Use it when a parameter is sensitive at *this* call site.

### 2. A `PrettyString()` method on your own type

If a type is *always* secret, make redaction intrinsic to it:

```go
type APIKey string

func (APIKey) PrettyString() string { return "***REDACTED***" }
```

Now every `APIKey` is redacted everywhere it is formatted, with nothing needed
at the call site.

### 3. A `PrintFuncFor` hook on the `Printer`

For types you do not own, or global rules (a struct tag, a value pattern, a
third-party type), install a hook:

```go
errs.Printer = errs.Printer.WithPrintFuncFor(func(v reflect.Value) pretty.PrintFunc {
    if v.Kind() == reflect.String && looksSensitive(v.String()) {
        return func(w io.Writer) (int, error) { return io.WriteString(w, "`***REDACTED***`") }
    }
    return pretty.PrintFuncForPrintable(v) // fall back to default
})
```

The hook runs for *every* value before the interface checks, so it can override
or extend the defaults. Returning `pretty.PrintFuncForPrintable(v)` for values
you do not handle preserves the standard behaviour.

### Why nesting is covered

go-pretty walks composite values recursively and applies the same interface
checks (and the same hook) to each field. So a secret nested inside a struct you
pass as a parameter is still redacted:

```
ProcessData(User{Name: `ann`, Password: ***REDACTED***})
                                        └─ redacted even though User was passed whole
```

You do not have to intercept the outer struct — redaction attaches to the
sensitive leaf wherever it appears.

## Trade-offs

- **Opt-in, not opt-out.** Nothing is redacted by default. A value leaks unless
  someone wrapped it, gave its type a `PrettyString`, or installed a hook. The
  `PrintFuncFor` hook exists to convert that into an opt-out policy (redact
  anything matching a rule) when you need defense in depth.
- **`KeepSecret` marks the call site, not the type.** It is precise but must be
  repeated at each call. That precision is also why
  [`go-errs-wrap replace`](../reference/go-errs-wrap.md) preserves `KeepSecret`
  wrappers rather than regenerating them away: the tool cannot infer which
  parameters are sensitive, so it keeps the annotation you wrote. An omitted
  parameter, by contrast, is re-added — which is why wrapping beats omitting.
- **The hook runs on every value.** A `PrintFuncFor` that does expensive work
  per value adds cost to every rendered error. Keep it a cheap predicate and let
  the default path handle the rest.
- **Redaction is a formatting concern, not encryption.** `Secret()` still
  returns the real value, and the value is still in memory and passed to the
  wrapped function. This protects logs and error trackers, not the process.

## Related

- [redact-sensitive-parameters.md](../how-to/redact-sensitive-parameters.md) — the three recipes in task form
- [configuration.md](../reference/configuration.md#printer) — the `Printer` variable and `WithPrintFuncFor`
- [api.md](../reference/api.md#secrets) — `Secret` and `KeepSecret`
