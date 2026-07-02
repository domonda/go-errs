# How to handle not-found and context errors

Detect "resource missing" and "context canceled / timed out" conditions
uniformly, including the standard-library variants, without scattering
`errors.Is` chains through your code.

## Prerequisites

- Go 1.24 or newer, with `github.com/domonda/go-errs` installed.

## Not-found errors

The helpers treat `errs.ErrNotFound`, `sql.ErrNoRows`, and `os.ErrNotExist` as
the same "not found" condition, so a database miss and a filesystem miss check
the same way.

1. Return `errs.ErrNotFound` (or a wrapper of it) when a lookup finds nothing:

   ```go
   var ErrUserNotFound = fmt.Errorf("user %w", errs.ErrNotFound)

   func GetUser(id string) (*User, error) {
       user, err := db.QueryUser(id)
       if errs.IsErrNotFound(err) {
           return nil, ErrUserNotFound
       }
       return user, err
   }
   ```

2. Check for any not-found variant with `IsErrNotFound`:

   ```go
   if errs.IsErrNotFound(err) {
       // handle the missing resource (e.g. return 404)
   }
   ```

3. Distinguish a *real* failure from an expected miss with
   `IsOtherThanErrNotFound`:

   ```go
   if errs.IsOtherThanErrNotFound(err) {
       return err // a genuine error, not just "absent"
   }
   ```

4. Swap any not-found variant for a domain error with `ReplaceErrNotFound`:

   ```go
   err = errs.ReplaceErrNotFound(err, ErrUserNotFound)
   ```

Declare package sentinels with `errs.Sentinel` so they can be `const`:

```go
const ErrOrderNotFound errs.Sentinel = "order not found"
```

## Context errors

The context helpers check the `Done` channel without blocking, so they are safe
to call anywhere.

1. Before or after a cancelable operation, branch on the specific condition:

   ```go
   if errs.IsContextCanceled(ctx) {
       return // caller went away
   }
   if errs.IsContextDeadlineExceeded(ctx) {
       return // timed out
   }
   ```

   Use `errs.IsContextDone(ctx)` when you do not care which of the two it was.

2. Avoid retrying on a context error you already hold:

   ```go
   if errs.IsContextError(err) {
       return err // do not retry a canceled/timed-out operation
   }
   ```

## Verification

- Not-found: call your lookup with an id you know is absent and confirm
  `IsErrNotFound` returns `true`; call it with a valid id and confirm
  `IsOtherThanErrNotFound` returns `false` on the `nil` error.
- Context: cancel a `context.WithCancel` and confirm `IsContextCanceled(ctx)`
  becomes `true`; let a `context.WithTimeout` expire and confirm
  `IsContextDeadlineExceeded(ctx)` becomes `true`.

## Troubleshooting

- **`errors.Is(err, ErrNotFound)` misses `sql.ErrNoRows`.** That is expected —
  plain `errors.Is` only checks the one target. Use `errs.IsErrNotFound`, which
  checks all three.
- **A context helper returns `false` for a canceled context.** `IsContextError`
  takes the *error*, while `IsContextCanceled`/`IsContextDone` take the
  *context*. Pass the right argument for the helper.

## Related

- [api.md](../reference/api.md#not-found-errors) — not-found helpers
- [api.md](../reference/api.md#context-errors) — context helpers
