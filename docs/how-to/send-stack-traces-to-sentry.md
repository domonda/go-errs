# How to send stack traces to Sentry

Get the real origin call stack of a `go-errs`-wrapped error into your Sentry
events, with no glue code and no dependency on `go-errs` from Sentry's side.

## Prerequisites

- Go 1.24 or newer, with `github.com/domonda/go-errs` installed.
- `sentry-go` set up in your service (`github.com/getsentry/sentry-go`), with
  `sentry.Init(...)` already called.
- Errors created or wrapped through this package (`New`, `Errorf`,
  `WrapWithCallStack`, or any `WrapWith*FuncParams`). A plain `errors.New` error
  has no stack to send.

## Steps

1. Produce errors through `errs`, as you already do for call-stack capture:

   ```go
   func ProcessUser(userID string) (err error) {
       defer errs.WrapWithFuncParams(&err, userID)
       return db.Update(userID) // wrapped with a call stack on failure
   }
   ```

2. Capture the error with Sentry as usual. No adapter, no conversion:

   ```go
   if err := ProcessUser(id); err != nil {
       hub.CaptureException(err)
   }
   ```

   `sentry.ExtractStacktrace` reflects on the error, finds the
   `StackTrace() []uintptr` method that every wrapped error exposes, and
   attaches the captured frames to the event.

3. Open the issue in Sentry and confirm the stack trace points at the real wrap
   sites (`ProcessUser`, its callers) rather than the SDK call site.

## Verification

Trigger a wrapped error on a code path you can find in Sentry, then check that
the event's stack trace shows your function and file/line where the error was
produced. If you have `debug=true` in `sentry.Init`, the SDK also logs whether a
stack trace was extracted.

## Troubleshooting

- **The Sentry event has no stack, or only the SDK's stack.** The error was not
  wrapped by `go-errs`. Ensure it flows through `New`/`Errorf`/`WrapWith*` and
  that nothing downstream replaced it with a plain error. `errors.Is`/`As` still
  work through the wrappers, so wrapping does not break your matching.
- **Only the innermost frame shows.** The stack trace is only as deep as your
  wrapping. Add `defer errs.WrapWith*FuncParams` to intermediate functions (or
  run [`go-errs-wrap insert`](manage-wrapping-with-go-errs-wrap.md)) so each
  layer contributes a frame.
- **You do not want to import Sentry into the library.** You do not have to —
  the integration is one reflection-discovered method; `go-errs` imports nothing
  from `sentry-go`.

## Related

- [sentry-stack-trace-interop.md](../explanation/sentry-stack-trace-interop.md) — why `StackTrace() []uintptr` and how Sentry finds it
- [api.md](../reference/api.md#sentry-interop) — the interop note in the API reference
- [call-stacks-and-wrapper-types.md](../explanation/call-stacks-and-wrapper-types.md) — where the frames come from
