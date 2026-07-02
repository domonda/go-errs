# Call Stacks and Wrapper Types

This explains how `errs` captures a call stack, why it uses two wrapper types,
and how the `skip` parameter and wrapper reuse fit together. For the API itself
see [api.md](../reference/api.md).

## The problem

A bare Go error is a string. When one surfaces in a log five layers deep, you
know *what* went wrong but not *where* or *with which inputs*. The standard
answer is to hand-annotate every return:

```go
if err != nil {
    return fmt.Errorf("ProcessUser(%q, %d): %w", userID, age, err)
}
```

That is verbose, easy to forget, and drifts out of sync with the real
parameters. What you actually want is: the file and line where each error was
produced, the chain of functions it passed through, and the argument values at
each hop — captured automatically, and cheaply enough to leave on in
production.

## The approach

`errs` captures the call stack at the moment an error is wrapped and defers all
formatting until `Error()` is called. Two things make this cheap:

- **Program counters, not text.** Capture stores a `[]uintptr` from
  `runtime.Callers`. Resolving those to file/line/function names via
  `runtime.CallersFrames` only happens when the error is rendered, so an error
  that is created and then handled without printing costs almost nothing.
- **A fixed-size buffer.** Capture allocates one slice of
  [`MaxCallStackFrames`](../reference/configuration.md#maxcallstackframes)
  (default 32) and slices it down to the frames actually filled.

```
error created/wrapped                 error printed (Error())
────────────────────                  ───────────────────────
runtime.Callers  ──►  []uintptr  ──►  runtime.CallersFrames  ──►  "func\n    file:line"
   (cheap, eager)     (stored)             (lazy, on demand)
```

### Two wrapper types

There are two internal wrappers, layered by embedding:

```
withCallStack                    withCallStackFuncParams
├── err        error             ├── withCallStack   (embedded)
└── callStack  []uintptr         │   ├── err        error
                                 │   └── callStack  []uintptr
                                 └── params  []any
```

- `withCallStack` carries just the stack. It backs `New`, `Errorf`,
  `WrapWithCallStack`, and `WrapWithCallStackSkip`.
- `withCallStackFuncParams` embeds `withCallStack` and adds the captured
  parameter values. It backs the `WrapWith*FuncParams` family.

Because `withCallStackFuncParams` embeds `withCallStack`, every method on the
stack wrapper (including `Unwrap`, `CallStack`, and `StackTrace`) is promoted to
it. So both wrappers behave identically for unwrapping and for
[Sentry discovery](sentry-stack-trace-interop.md); the params wrapper just adds
argument rendering on top.

### Rendering order

`formatError` unwraps the chain from outer to inner, collects one formatted
frame per wrapper, then prints the root message followed by those frames in
reverse. The result reads innermost-first, matching how you read a stack trace:

```
error in funcC                 ← root message
main.funcC()                   ← innermost wrap
    …/wrapping.go:25
main.funcB([`Hello World!`])   ← middle wrap, with its parameter
    …/wrapping.go:19
main.funcA(666, `Hello World!`)← outermost wrap, with its parameters
    …/wrapping.go:13
```

Each wrapper contributes exactly the **top** frame it captured — the line where
that function returned the error — not its whole stack. The chain of wrap points
*is* the stack trace.

### The `skip` parameter

`runtime.Callers` reports the stack from wherever it is called, which is inside
the library, not inside your code. `skip` counts how many frames to drop so the
first reported frame is your call site. The direct entry points bake in the
right value (`WrapWithFuncParams` uses `skip=0` relative to its caller); the
`*Skip` variants expose it so you can build your own wrapper helpers:

```go
// One extra frame of indirection → skip must increase by one to compensate.
func myWrapper(skip int, err error) error {
    return errs.WrapWithCallStackSkip(1+skip, err)
}
```

Get `skip` wrong and the stack points one frame above or below the real site.

### Reusing an existing stack

If you call a `WrapWith*FuncParams` function on an error that is *already* a
call-stack wrapper, it does not capture a second stack. It reuses the existing
program counters and only attaches the parameters:

```
err is *withCallStack  ──►  new *withCallStackFuncParams reusing err's callStack + your params
err is not wrapped     ──►  new *withCallStackFuncParams capturing a fresh stack + your params
```

This keeps the innermost capture (closest to the failure) as the source of
truth for the stack, while still letting an outer function record its argument
values.

## Trade-offs

- **Fixed-size capture buffer.** Every wrap allocates
  `[]uintptr` of length `MaxCallStackFrames`, even for a shallow stack. That is
  the cost of not walking the stack twice. Tune the depth to your call graph.
- **Program counters are build-specific.** Stored PCs are only meaningful for
  the running binary; they are not serializable across builds. `StackTrace()`
  exists precisely so external tools resolve them in-process.
- **One frame per wrapper, not the full stack.** The rendered trace is only as
  detailed as your wrapping is dense. A function with no `defer errs.Wrap*`
  contributes no line — which is exactly what
  [`go-errs-wrap`](../reference/go-errs-wrap.md) automates away.
- **Lazy formatting hides cost until print time.** Rendering a deep chain does
  real work (symbol resolution, go-pretty formatting). That is the right place
  for it — you only pay when you actually surface the error.

## Related

- [api.md](../reference/api.md) — the wrapping functions
- [configuration.md](../reference/configuration.md) — `MaxCallStackFrames`, `TrimFilePathPrefix`
- [sentry-stack-trace-interop.md](sentry-stack-trace-interop.md) — how the stored PCs reach Sentry
