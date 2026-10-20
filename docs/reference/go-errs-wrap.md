# `go-errs-wrap` CLI Reference

`go-errs-wrap` is a code-transformation tool that manages
`defer errs.Wrap*` statements in Go source files. It can remove them, replace
them with freshly generated statements that match each function's current
parameters, or insert them into functions that lack one.

## Installation

`go-errs-wrap` is the module `github.com/domonda/go-errs/cmd/go-errs-wrap`
and needs `go-errs` v1.1.0 or newer in your module, so add `go-errs` first:

```bash
go get github.com/domonda/go-errs@latest
go get -tool github.com/domonda/go-errs/cmd/go-errs-wrap@latest
```

Run it with `go tool go-errs-wrap`, or install the `go-errs-wrap` binary
from within your module with `go install github.com/domonda/go-errs/cmd/go-errs-wrap`.
Older `go-errs` versions still contain the tool's packages, which makes its
import ambiguous, and without a `go-errs` requirement the tool's own
requirement on `go-errs` can't be resolved.
`go install github.com/domonda/go-errs/cmd/go-errs-wrap@latest` does not work:
the tool's module replaces `go-errs` with its local checkout.

To keep the tool's dependencies, like `golang.org/x/tools`, out of your
module's `go.mod`, add the tool to a separate tools module file instead:

```bash
mkdir tools && (cd tools && go mod init example.com/yourmodule/tools)
go get -modfile=tools/go.mod github.com/domonda/go-errs@latest
go get -modfile=tools/go.mod -tool github.com/domonda/go-errs/cmd/go-errs-wrap@latest
go tool -modfile=tools/go.mod go-errs-wrap insert ./...
```

## Synopsis

```
go-errs-wrap <command> [options] <path>
```

`<command>` is one of `remove`, `replace`, or `insert`. Options must come
before `<path>`; any argument after it is an error. `<path>` is a single
`.go` file or a directory. A directory processes the Go package in it. A trailing
`/...` (or a bare `...`) processes all packages below the directory.
Files are selected like the `go` command selects them: `main` packages are
included, while a directory skips files that need custom build tags and files
whose names start with `.` or `_`, which are only processed when passed as
`<path>` themselves. A trailing `/...` also skips `testdata`, `vendor` and
`node_modules` directories, directories starting with `.` or `_`, directories
that the module's `go.mod` ignores, and nested modules. Unlike the `go` command,
`/...` on a directory that is not within a module, like the root of a repository
with several modules and a `go.work` file but no `go.mod`, processes all modules
below it; `go.work` is not read.

## Commands

| Command   | Description                                                        |
| --------- | ----------------------------------------------------------------- |
| `remove`  | Remove all `defer errs.Wrap*` statements and `//#wrap-result-err` marker comments |
| `replace` | Replace existing `defer errs.Wrap*` statements and `//#wrap-result-err` markers with freshly generated code that matches each function's parameters |
| `insert`  | Insert a `defer errs.Wrap*` at the first line of every function that has a named error result and does not already have one (followed by a blank line) |

## Options

| Option          | Description                                          |
| --------------- | --------------------------------------------------- |
| `-out <path>`   | Write results to `<path>` instead of modifying the source in place. A directory source produces a copy of the directory, or of the whole directory tree with `/...`, in which only the rewritten Go files differ; version control metadata (`.git`, `.hg`, `.svn`), symbolic links to directories and special files are not copied, and `<path>` must not be the source directory, be within it or contain it. Symbolic links to directories in an existing `<path>` must not lead outside of it. An empty `<path>` is an error. A file source is written to `<path>` if it ends in `.go`, otherwise into the directory `<path>`, which is created if missing. Ignored when `-validate` is set. |
| `-minvariadic`  | Always emit the specialized `WrapWithNFuncParams` variant instead of preserving an existing variadic `WrapWithFuncParams` call |
| `-validate`     | Dry-run: modify nothing, report issues to stderr, exit `1` if any are found. For CI |
| `-verbose`      | Print progress to stdout                            |
| `-help`         | Show usage and exit                                 |

Only files ending in `.go` are transformed; `*_test.go` files are never
processed, also not when passed as `<path>`.

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
| `2`  | Invalid option, like an unknown flag or a flag without its value |

## Generated statement selection

`replace` and `insert` choose the wrap function from the parameter count:

- 0 parameters → `defer errs.WrapWith0FuncParams(&err)`
- 1 parameter → `defer errs.WrapWith1FuncParam(&err, p0)`
- 2–10 parameters → `defer errs.WrapWithNFuncParams(&err, …)`
- 11+ parameters → variadic `defer errs.WrapWithFuncParams(&err, …)`

`&err` uses the function's actual named error result. A missing
`github.com/domonda/go-errs` import is added automatically and the imports are
then grouped with `goimports`. Otherwise the rewritten file is formatted like
`gofmt`, keeping its existing import groups.

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

# Write results to a separate directory tree instead of editing in place
go-errs-wrap insert -out ./output ./pkg/...

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
