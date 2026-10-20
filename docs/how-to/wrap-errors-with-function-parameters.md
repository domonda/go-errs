# How to wrap errors with function parameters

Capture the call stack and the argument values of a function every time it
returns an error, so failures carry their context into your logs.

## Prerequisites

- Go 1.24 or newer.
- The package installed: `go get github.com/domonda/go-errs`.
- A function whose error return is a **named** result (`(err error)`), because
  the wrap functions take a pointer to it.

## Steps

1. Give the function a named error result and `defer` the wrap on its first
   line, passing every parameter you want recorded:

   ```go
   import "github.com/domonda/go-errs"

   func ProcessUser(userID string, age int) (err error) {
       defer errs.WrapWithFuncParams(&err, userID, age)

       if age < 0 {
           return errors.New("invalid age")
       }
       return database.UpdateUser(userID, age)
   }
   ```

   The wrap is a no-op when the function returns `nil`, so it is safe to leave
   on every path.

2. Return errors normally. Any non-nil error is wrapped with the stack frame and
   the parameter values at return time. A returned error renders as:

   ```
   invalid age
   main.ProcessUser(`user-123`, -5)
       /path/to/file.go:45
   ```

3. For a hot function with a fixed parameter count (0–10), switch to the
   numbered variant to avoid the variadic slice allocation:

   ```go
   defer errs.WrapWith2FuncParams(&err, userID, age)
   ```

   Use `WrapWith0FuncParams(&err)` for none and `WrapWith1FuncParam(&err, p0)`
   for one (note the singular name). For 11+ parameters, stay on
   `WrapWithFuncParams`.

4. Let it stack across call layers. Each wrapped function contributes one line,
   innermost first:

   ```
   error in funcC
   main.funcC()
       …/wrapping.go:25
   main.funcB([`Hello World!`])
       …/wrapping.go:19
   main.funcA(666, `Hello World!`)
       …/wrapping.go:13
   ```

## Verification

Run the bundled example, which wraps three nested calls:

```bash
go run github.com/domonda/go-errs/examples/wrapping@latest
```

You should see the three-frame trace above. In your own code, trigger the error
path and confirm the log line shows `FunctionName(params…)` followed by
`file:line`.

## Troubleshooting

- **Parameters do not appear, only a bare message.** The error was not wrapped.
  Check that the `defer` is present and that the result is *named* (`(err
  error)`, not `error`). An unnamed result cannot be taken by pointer.
- **The stack points one line off.** You are wrapping through a helper. Use
  `WrapWithFuncParamsSkip(skip, &err, …)` and raise `skip` by one per extra
  layer. See
  [call-stacks-and-wrapper-types.md](../explanation/call-stacks-and-wrapper-types.md).
- **A sensitive value is printed.** Wrap it with `errs.KeepSecret(...)`. See
  [redact-sensitive-parameters.md](redact-sensitive-parameters.md).
- **Writing these by hand across a package is tedious.** Generate them with
  [`go-errs-wrap insert`](manage-wrapping-with-go-errs-wrap.md).

## Related

- [api.md](../reference/api.md#wrapping-with-function-parameters) — the full function list
- [Getting started tutorial](../tutorials/getting-started.md)
