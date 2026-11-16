# How to return error messages to clients without call stacks

Send an error's message to a client (an HTTP response, a JSON error body, an
MCP tool result) with all its context but without the call stacks and function
parameters that `errs` adds for your logs.

## Prerequisites

- Go 1.26 or newer, with `github.com/domonda/go-errs` v1.3.0 or newer
  installed (`MessageWithoutCallStacks` was added in v1.3.0).
- Errors created or wrapped with `errs` (`New`, `Errorf`, `WrapWithCallStack`,
  `WrapWithFuncParams`, the `Recover*` helpers).

## Why the plain message is not enough

`err.Error()` of an `errs` error ends with the call stack, and wrappers from
`WrapWithFuncParams` add the parameter values. With the functions from step 1
below, `createOrder(Order{Quantity: 0})` returns an error whose message is:

```
create order: invalid order: quantity must be positive, got 0
main.validateOrder(Order{Quantity:0})
    /src/shop/order.go:22

main.validateOrder(Order{Quantity:0})
    /src/shop/order.go:22
main.createOrder
    /src/shop/order.go:29
```

The first block is the inner error's call stack, which `errs.Errorf("create
order: %w", err)` copied into the outer message; the second is the outer
error's. That is what you want in a log. Sent to a client, it leaks source
paths, line numbers and argument values. `UnwrapCallStack` removes only the
second block. `errs.MessageWithoutCallStacks(err)` removes both and returns
just the first line: `create order: invalid order: quantity must be positive,
got 0`.

## Steps

1. Wrap errors with `%w`, never `%v` or `%s`, so `MessageWithoutCallStacks`
   can reach the inner errors and their call stacks:

   ```go
   const ErrInvalidOrder errs.Sentinel = "invalid order"

   func validateOrder(o Order) (err error) {
       defer errs.WrapWithFuncParams(&err, o)

       if o.Quantity <= 0 {
           return errs.Errorf("%w: quantity must be positive, got %d", ErrInvalidOrder, o.Quantity)
       }
       return nil
   }

   func createOrder(o Order) error {
       if err := validateOrder(o); err != nil {
           return errs.Errorf("create order: %w", err)
       }
       // ...
       return nil
   }
   ```

   `errs.Errorf("create order: %v", err)` would copy the inner message,
   call stack included, into plain text that `MessageWithoutCallStacks` cannot
   tell apart from your own words.

2. Log the full error, then send the client `MessageWithoutCallStacks(err)`
   for errors the client caused and can fix:

   ```go
   func writeError(w http.ResponseWriter, err error) {
       log.Println(err) // full message with call stacks and parameters

       if errors.Is(err, ErrInvalidOrder) {
           http.Error(w, errs.MessageWithoutCallStacks(err), http.StatusBadRequest)
           return
       }
       http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
   }
   ```

   The client gets `create order: invalid order: quantity must be positive,
   got 0` with status 400.

3. Answer every other error with a generic message, as the `500` branch above
   does. `MessageWithoutCallStacks` removes only what `errs` adds; the message
   text of other errors stays as written. For example, an `os.Open` failure
   wrapped as `errs.Errorf("load config: %w", err)` still reads
   `load config: open /etc/app/config.json: no such file or directory`, and
   database driver errors can contain table names or query fragments. Decide per
   error class what the client may see, as in step 2, and do not send the
   message of an unexpected error.

4. Send any other client-facing text the same way, for example a JSON error
   body:

   ```go
   w.Header().Set("Content-Type", "application/json")
   w.WriteHeader(http.StatusBadRequest)
   json.NewEncoder(w).Encode(map[string]string{
       "error": errs.MessageWithoutCallStacks(err),
   })
   ```

5. Treat recovered panics as server errors. `MessageWithoutCallStacks` removes
   the goroutine dump that `RecoverPanicAsError` and `AsErrorWithDebugStack`
   append, but the panic value itself, such as
   `runtime error: index out of range [3] with length 0`, describes your code,
   not the client's request. Send it through the generic branch of step 3.

## Verification

Test the client message of each error the client is allowed to see:

```go
func TestCreateOrderClientMessage(t *testing.T) {
    err := createOrder(Order{Quantity: 0})

    msg := errs.MessageWithoutCallStacks(err)
    if msg != "create order: invalid order: quantity must be positive, got 0" {
        t.Errorf("unexpected client message: %q", msg)
    }
    if strings.Contains(msg, ".go:") {
        t.Errorf("client message contains a call stack: %q", msg)
    }
}
```

Every rendered call-stack frame contains a `.go:` file and line, so the second
check catches a stack that slipped through.

## Troubleshooting

- **The client message still contains a call stack.** An error on the way was
  formatted with `%v` or `%s` instead of `%w`. Search for
  `Errorf("…: %v", err)` patterns and switch them to `%w`.
- **A call stack survives inside a custom wrapper's text.**
  `MessageWithoutCallStacks` finds a wrapped error's call stack by searching
  for that error's exact `Error()` text. A wrapper that reformats the inner
  message (trims, quotes, upper-cases it) or lists errors that its `Unwrap()`
  does not return, like `hashicorp/go-multierror`, hides it. Make the wrapper
  include `inner.Error()` verbatim and return its errors from
  `Unwrap() error` or `Unwrap() []error`, or use `errors.Join`.
- **The message contains `%!v(PANIC=Error method: …)`.** An error's `Error()`
  method panicked, for example on a nil pointer or nil field.
  `MessageWithoutCallStacks` does not panic, it uses the same text `fmt` renders
  for such an error. Fix the error value at the place it is created.
- **The message shows an old parameter value.** Each `errs` error renders its
  message on the first `Error()` call and caches it, so later changes to a
  parameter (a canceled `context.Context`, a modified struct behind a pointer)
  do not appear. This keeps the text identical to what wrapping errors
  copied. See
  [call-stacks-and-wrapper-types.md](../explanation/call-stacks-and-wrapper-types.md#trade-offs).

## Related

- [api.md](../reference/api.md#func-messagewithoutcallstackserr-error-string) —
  `MessageWithoutCallStacks` reference, and how it compares with
  `UnwrapCallStack` and `Root`
- [wrap-errors-with-function-parameters.md](wrap-errors-with-function-parameters.md)
- [recover-panics-as-errors.md](recover-panics-as-errors.md)
- [call-stacks-and-wrapper-types.md](../explanation/call-stacks-and-wrapper-types.md) —
  why the message is cached
