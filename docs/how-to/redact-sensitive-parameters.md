# How to redact sensitive parameters

Keep passwords, tokens, keys, and other secrets out of error call stacks while
still recording that the parameter was there. Pick the recipe that matches how
local or global the secret is.

## Prerequisites

- Go 1.24 or newer, with `github.com/domonda/go-errs` installed.
- A function that wraps errors with parameters (see
  [wrap-errors-with-function-parameters.md](wrap-errors-with-function-parameters.md)).

## Recipe 1: redact one parameter at the call site (`KeepSecret`)

Use when a parameter is sensitive *here*.

1. Wrap the value with `errs.KeepSecret` in the wrap call:

   ```go
   func Login(username, password string) (err error) {
       defer errs.WrapWithFuncParams(&err, username, errs.KeepSecret(password))
       return authenticate(username, password)
   }
   ```

2. Errors now render the secret as a placeholder:

   ```
   Login(`admin`, ***REDACTED***)
   ```

Prefer this over dropping the parameter from the wrap call:
[`go-errs-wrap replace`](manage-wrapping-with-go-errs-wrap.md) preserves
`KeepSecret` wrappers but re-adds omitted parameters.

## Recipe 2: make a type always redact (`PrettyString`)

Use when a type is *always* secret, so you never have to remember at the call
site.

1. Implement `PrettyString()` on the type:

   ```go
   type APIKey string

   func (APIKey) PrettyString() string { return "***REDACTED***" }
   ```

2. Pass it normally; it redacts everywhere it is formatted, including when nested
   inside a struct:

   ```go
   defer errs.WrapWithFuncParams(&err, apiKey) // renders: fn(***REDACTED***)
   ```

## Recipe 3: redact by global rule (`PrintFuncFor`)

Use for types you do not own, struct-tag policies, or value patterns.

1. Install a hook on the `Printer` at startup, and always fall back to the
   default for values you do not handle:

   ```go
   func init() {
       errs.Printer = errs.Printer.WithPrintFuncFor(func(v reflect.Value) pretty.PrintFunc {
           // Redact struct fields tagged `secret:"true"`.
           if v.Kind() == reflect.Struct {
               t := v.Type()
               for i := range t.NumField() {
                   if t.Field(i).Tag.Get("secret") == "true" {
                       return func(w io.Writer) (int, error) {
                           return io.WriteString(w, t.Name()+"{***FIELDS_REDACTED***}")
                       }
                   }
               }
           }
           return pretty.PrintFuncForPrintable(v) // default handling
       })
   }
   ```

2. Tag the sensitive fields:

   ```go
   type User struct {
       ID       string
       Password string `secret:"true"`
   }
   ```

## Verification

Trigger an error path that includes the secret parameter and confirm the log
shows `***REDACTED***` (or your placeholder) instead of the value. For Recipe 2,
also nest the type inside a struct parameter and confirm it is still redacted.

## Troubleshooting

- **Still printing the real value with `KeepSecret`.** Make sure you passed
  `errs.KeepSecret(x)` into the wrap call, not `x` — and that you did not also
  pass `x` unwrapped elsewhere in the same call.
- **`PrintFuncFor` broke formatting of other values.** Your hook must return
  `pretty.PrintFuncForPrintable(v)` for anything it does not handle; otherwise
  those values lose default formatting.
- **`replace` removed my redaction.** It only preserves `KeepSecret` wrappers.
  If you had simply omitted the parameter, wrap it with `KeepSecret` instead.

## Related

- [secret-redaction-and-pretty-printing.md](../explanation/secret-redaction-and-pretty-printing.md) — why this works and the interface priority order
- [configuration.md](../reference/configuration.md#printer) — the `Printer` and `WithPrintFuncFor`
- [api.md](../reference/api.md#secrets) — `Secret`, `KeepSecret`
