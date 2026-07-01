# How to manage wrapping with go-errs-wrap

Add, update, or strip `defer errs.Wrap*` statements across a whole codebase
mechanically instead of by hand, and enforce their presence in CI.

## Prerequisites

- Go 1.24 or newer.
- The tool installed:

  ```bash
  go install github.com/domonda/go-errs/cmd/go-errs-wrap@latest
  ```

- A version-controlled working tree — these commands rewrite files in place by
  default, so commit or stash first.

## Insert wrap statements into functions that lack them

1. Run `insert` over a package (recurse with `/...`):

   ```bash
   go-errs-wrap insert ./...
   ```

   For every function with a named error result and no existing wrap, it inserts
   the right `defer errs.WrapWith*FuncParams(&err, …)` on the first line,
   followed by a blank line, and adds the `go-errs` import.

2. Review the diff and build:

   ```bash
   git diff
   go build ./...
   ```

## Replace outdated wrap statements

When you add or remove function parameters, the existing wrap call drifts out of
date. Regenerate them:

```bash
go-errs-wrap replace ./...
```

`replace` also expands `//#wrap-result-err` marker comments into real
statements, and **preserves** any parameter already wrapped in
`errs.KeepSecret(...)`. Drop a marker where you want a wrap generated:

```go
func LoadFile(path string) (data []byte, err error) {
    //#wrap-result-err
    return nil, nil
}
```

## Remove all wrap statements

```bash
go-errs-wrap remove ./...
```

This deletes every `defer errs.Wrap*` statement and `//#wrap-result-err` marker.

## Useful flags

- `-out ./somewhere` — write results to a separate tree instead of editing in
  place (ignored with `-validate`).
- `-minvariadic` — always emit the specialized `WrapWithNFuncParams` variant
  rather than preserving an existing variadic call.
- `-verbose` — print each change as it is made.

## Enforce wrapping in CI with `-validate`

`-validate` changes nothing; it reports issues to stderr and exits `1` if any
are found. Add a step to your pipeline:

```bash
# Fail if any function with a named error result is missing a wrap statement
go-errs-wrap insert -validate ./...
```

You can validate the other modes too: `replace -validate` fails if any wrap is
out of date; `remove -validate` fails if any wrap still exists.

## Verification

- After `insert`/`replace`, run `go build ./...` and `go vet ./...`; the tool
  re-sorts imports with `goimports`, so the result should compile cleanly.
- Run the matching `-validate` command and confirm it exits `0` once the tree is
  in the desired state.

## Troubleshooting

- **A function was skipped by `insert`.** It either has no named error result or
  already has a wrap statement — both are skipped by design. Name the result
  (`(err error)`) to opt it in.
- **`replace` re-added a parameter I removed on purpose.** The tool cannot know
  a parameter is sensitive. Wrap it with `errs.KeepSecret(...)` (which `replace`
  preserves) instead of omitting it. See
  [redact-sensitive-parameters.md](redact-sensitive-parameters.md).
- **A `//#wrap-result-err` marker was left untouched with a warning.** It was
  not inside a function body. Move it onto its own line inside the function.
- **`_test.go` files were not changed.** Test files are skipped when processing a
  directory; point the tool at the file directly if you need it.

## Related

- [go-errs-wrap.md](../reference/go-errs-wrap.md) — full command, flag, and exit-code reference
- [wrap-errors-with-function-parameters.md](wrap-errors-with-function-parameters.md) — what the generated statements do
