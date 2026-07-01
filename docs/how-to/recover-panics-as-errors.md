# How to recover panics as errors

Turn a panic into a returned `error` (or a logged one) so a single bad path does
not crash the process, and so the panic carries a stack and the function's
parameters.

## Prerequisites

- Go 1.24 or newer, with `github.com/domonda/go-errs` installed.
- A function with a **named** error result for the recover-into-result helpers.

## Recover a panic into the error result

1. `defer errs.RecoverPanicAsError(&err)` on a function that might panic:

   ```go
   func RiskyOperation() (err error) {
       defer errs.RecoverPanicAsError(&err)
       return doSomethingRisky() // may panic
   }
   ```

   If the function panics, `err` is set to the panic value converted to an
   error and wrapped with `debug.Stack()`. If the function was already returning
   a non-nil error when the panic happened, the two are combined into
   `function returning error (…) panicked with: …`.

2. To also record the function's parameters, use the `WithFuncParams` variant:

   ```go
   func ProcessItem(id string) (err error) {
       defer errs.RecoverPanicAsErrorWithFuncParams(&err, id)
       return processItem(id) // may panic
   }
   ```

## Log a panic instead of (or as well as) returning it

1. Log and re-panic — for observability at a boundary you do not want to
   swallow:

   ```go
   defer errs.LogPanicWithFuncParams(logger, id) // logs, then re-panics
   ```

2. Log and continue — recover without re-panicking:

   ```go
   defer errs.RecoverAndLogPanicWithFuncParams(logger, id) // logs, does not re-panic
   ```

   Both take a [`Logger`](../reference/api.md#type-logger-interface) (anything
   with `Printf(format string, args ...any)`, such as `*log.Logger`).

## Convert an arbitrary recovered value yourself

If you run your own `recover()`, convert the value with `AsError` (no stack) or
`AsErrorWithDebugStack` (adds `debug.Stack()`):

```go
if p := recover(); p != nil {
    err = errs.AsErrorWithDebugStack(p)
}
```

`AsError` handles `nil`, `error`, `[]error` (joined), `string`, and
`fmt.Stringer`, and falls back to `fmt.Sprint` for anything else.

## Verification

Write a function that panics (e.g. a nil-map write), guard it with
`RecoverPanicAsError`, and confirm the caller receives a non-nil `error` whose
message includes the panic value and a stack — and that the process keeps
running.

## Troubleshooting

- **The panic still crashes the program.** The `defer` must be in the same
  function that panics (or an ancestor on the call path), and `recover` only
  works inside a deferred call. Confirm the `defer errs.Recover…` line runs
  before the panic.
- **`err` is nil after a panic.** The result must be *named* (`(err error)`);
  the helper assigns through the pointer you pass. An unnamed result cannot be
  updated.
- **`LogPanicWithFuncParams` crashed my service.** It re-panics by design. Use
  `RecoverAndLogPanicWithFuncParams` to log without re-panicking.

## Related

- [api.md](../reference/api.md#panic-recovery) — the full panic-recovery API
- [wrap-errors-with-function-parameters.md](wrap-errors-with-function-parameters.md)
