# `go-errs-wrap` CLI Reference

`go-errs-wrap` is a code-transformation tool that manages
`defer errs.Wrap*` statements in Go source files. It can remove them, replace
them with freshly generated statements that match each function's current
parameters, or insert them into functions that lack one.

## Installation

```bash
go install github.com/domonda/go-errs/cmd/go-errs-wrap@latest
```

## Synopsis

```
go-errs-wrap <command> [options] <path>
```

`<command>` is one of `remove`, `replace`, or `insert`. `<path>` is a single
`.go` file or a directory. A trailing `/...` (or a bare `...`) processes the
directory recursively.

## Commands

| Command   | Description                                                        |
| --------- | ----------------------------------------------------------------- |
| `remove`  | Remove all `defer errs.Wrap*` statements and `//#wrap-result-err` marker comments |
| `replace` | Replace existing `defer errs.Wrap*` statements and `//#wrap-result-err` markers with freshly generated code that matches each function's parameters |
| `insert`  | Insert a `defer errs.Wrap*` at the first line of every function that has a named error result and does not already have one (followed by a blank line) |

## Options

| Option          | Description                                          |
| --------------- | --------------------------------------------------- |
| `-out <path>`   | Write results to `<path>` instead of modifying the source in place. A directory source produces a copied directory tree; non-Go files are copied unchanged. Ignored when `-validate` is set. |
| `-minvariadic`  | Always emit the specialized `WrapWithNFuncParams` variant instead of preserving an existing variadic `WrapWithFuncParams` call |
| `-validate`     | Dry-run: modify nothing, report issues to stderr, exit `1` if any are found. For CI |
| `-verbose`      | Print progress to stdout                            |
| `-help`         | Show usage and exit                                 |

Only files ending in `.go` are transformed; `*_test.go` files are skipped when
processing a directory.

### What `-validate` checks per command

| Command   | `-validate` reports                                    |
| --------- | ------------------------------------------------------ |
| `remove`  | any `defer errs.Wrap*` statements or markers still present |
| `replace` | any wrap statements whose parameters are out of date   |
| `insert`  | any functions with a named error result but no wrap    |

## Exit codes

| Code | Meaning                                                         |
| ---- | -------------------------------------------------------------- |
| `0`  | Success — operation completed, or `-validate` found no issues   |
| `1`  | Error — validation found issues, or a bad argument / missing file |

## Generated statement selection

`replace` and `insert` choose the wrap function from the parameter count:

- 0 parameters → `defer errs.WrapWith0FuncParams(&err)`
- 1 parameter → `defer errs.WrapWith1FuncParam(&err, p0)`
- 2–10 parameters → `defer errs.WrapWithNFuncParams(&err, …)`
- 11+ parameters → variadic `defer errs.WrapWithFuncParams(&err, …)`

`&err` uses the function's actual named error result. The required
`github.com/domonda/go-errs` import is added automatically, and imports are
re-sorted with `goimports`.

## The `//#wrap-result-err` marker

Instead of writing a `defer errs.Wrap*` line by hand, you can drop a marker
comment on its own line inside a function:

```go
func LoadFile(path string) (data []byte, err error) {
    //#wrap-result-err
    return nil, nil
}
```

`replace` turns each marker into the correct statement for that function, and
`remove` deletes markers along with real wrap statements. A marker that is not
inside a function is left in place with a warning to stderr.

## Secret preservation

`replace` preserves parameters already wrapped in `errs.KeepSecret(...)`, so a
secret you mark once stays redacted across regenerations. Omitting a sensitive
parameter entirely does **not** survive — `replace` re-adds it — so wrap it with
`KeepSecret` instead. See
[redact-sensitive-parameters.md](../how-to/redact-sensitive-parameters.md).

## Examples

```bash
# Insert wrap statements into every function that is missing one
go-errs-wrap insert ./pkg/...

# Update the parameters of existing wrap statements in one file
go-errs-wrap replace ./pkg/mypackage/file.go

# Remove all wrap statements recursively
go-errs-wrap remove ./pkg/...

# Write results to a separate tree instead of editing in place
go-errs-wrap insert -out ./output ./pkg/mypackage

# CI: fail the build if any function is missing a wrap statement
go-errs-wrap insert -validate ./pkg/...
```

### Example transformation (`insert`)

Input:

```go
package example

func ProcessData(ctx context.Context, id string) (err error) {
    return doWork(ctx, id)
}
```

After `go-errs-wrap insert example.go`:

```go
package example

import "github.com/domonda/go-errs"

func ProcessData(ctx context.Context, id string) (err error) {
    defer errs.WrapWith2FuncParams(&err, ctx, id)

    return doWork(ctx, id)
}
```

`insert` skips functions without a named error result and functions that
already have a wrap statement.

## Related

- [manage-wrapping-with-go-errs-wrap.md](../how-to/manage-wrapping-with-go-errs-wrap.md) — task recipes and a CI setup
- [api.md](api.md) — the `WrapWith*FuncParams` functions the tool emits
