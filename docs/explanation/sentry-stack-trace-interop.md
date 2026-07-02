# Sentry Stack-Trace Interop

This explains why the wrapper types expose a `StackTrace() []uintptr` method,
how `sentry-go` finds it, and what the design deliberately avoids. For the task
recipe see
[send-stack-traces-to-sentry.md](../how-to/send-stack-traces-to-sentry.md).

## The problem

`errs` already captures a precise call stack for every wrapped error (see
[call-stacks-and-wrapper-types.md](call-stacks-and-wrapper-types.md)). But that
stack lives behind an unexported wrapper and a package-private
`CallStack() []uintptr` method. When you send such an error to Sentry with
`hub.CaptureException(err)`, Sentry has no idea the stack is there. The event
shows up with only the message, or with the shallow stack of the Sentry SDK
call site — not the real origin of the error.

Sentry cannot import this package and call `CallStack()`: that is not part of
any shared interface, and wiring per-project glue code to translate our stack
into Sentry's format is exactly the kind of boilerplate the library exists to
remove.

## How `sentry-go` discovers a stack

`sentry.ExtractStacktrace` does not depend on any one error library. It reflects
on the error's concrete type and looks for a method with one of three names,
established by three popular libraries:

```
method name        convention from
───────────        ───────────────
StackTrace()       pkg/errors
StackFrames()      go-errors/errors
GetStackTracer()   pingcap/errors
```

It then reads program counters out of the returned slice, accepting either a
bare `uintptr` element or a struct element exposing a `ProgramCounter` or `PC`
field.

## The approach

`errs` implements the simplest of those shapes — the `pkg/errors` one:

```go
func (w *withCallStack) StackTrace() []uintptr {
    return slices.Clone(w.callStack)
}
```

It returns the raw program counters already captured at wrap time, in the order
`runtime.Callers` produced them (innermost first). The slice is a copy, matching
the `pkg/errors` contract, so a consumer that reorders it in place cannot corrupt
the error's stored call stack. Because
`withCallStackFuncParams` embeds `withCallStack`, the method is promoted, so
both wrapper types satisfy the probe. Every error from `New`, `Errorf`,
`WrapWithCallStack`, and the `WrapWith*FuncParams` family is therefore picked up
by Sentry automatically:

```
errs.New / Errorf / WrapWith*  ──►  *withCallStack(FuncParams)
                                          │  StackTrace() []uintptr
                                          ▼
                          sentry.ExtractStacktrace (reflection)
                                          │
                                          ▼
                            Sentry event with the real call stack
```

No import of `sentry-go`, `pkg/errors`, or any other library is added to
`go-errs`. The whole integration is one method that happens to match a name
Sentry already probes for.

## Trade-offs

- **Reflection contract, not a compile-time one.** The link between our method
  and Sentry is a method name discovered by reflection. If `sentry-go` ever
  renamed or dropped `StackTrace`-style probing, this would silently stop
  working. A compile-time interface guard
  (`_ interface{ StackTrace() []uintptr } = &withCallStack{}`) protects *our*
  side of the contract, but not Sentry's expectation of the name.
- **Method-name overloading.** `StackTrace() []uintptr` exists purely for tool
  interop; `CallStack() []uintptr` remains the package's own accessor. Two
  methods return the same data under two names — a small redundancy that buys a
  zero-dependency integration.
- **`pkg/errors` shape, not the richer `Frame` shape.** Returning bare
  `uintptr`s (rather than resolved frames) is the least Sentry needs and keeps
  capture cheap. Sentry resolves the PCs itself, in-process, where they are
  still meaningful.
- **In-process only.** Program counters are meaningful only in the running
  binary. The integration works because Sentry resolves them before the process
  exits; the PCs are not portable across builds or machines.

## Related

- [call-stacks-and-wrapper-types.md](call-stacks-and-wrapper-types.md) — where the PCs come from
- [send-stack-traces-to-sentry.md](../how-to/send-stack-traces-to-sentry.md) — how to use it
- [api.md](../reference/api.md#sentry-interop) — the interop section of the API reference
