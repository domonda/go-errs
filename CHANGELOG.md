# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed

- `go-errs-wrap` is its own module `github.com/domonda/go-errs/cmd/go-errs-wrap`,
  so the `go-errs` module no longer depends on `github.com/ungerik/go-astvisit`
  and `golang.org/x/tools`. The tool needs `go-errs` v1.1.0 or newer in your
  module, so add it with two commands:
  `go get github.com/domonda/go-errs@latest`, then
  `go get -tool github.com/domonda/go-errs/cmd/go-errs-wrap@latest`.
  Because the tool's module replaces `go-errs` with its local checkout,
  `go install github.com/domonda/go-errs/cmd/go-errs-wrap@latest` no longer works.
- Upgrading a module that already has `tool github.com/domonda/go-errs/cmd/go-errs-wrap`:
  run `go get github.com/domonda/go-errs@latest` first, then `go mod tidy`.
  Until `go-errs` is upgraded, `go get -u tool` fails with an ambiguous import,
  because older `go-errs` versions still contain the tool's packages.
- `go-errs-wrap` selects files like the `go` command: `main` packages are now
  processed, while a directory path skips files that need custom build tags and
  files whose names start with `.` or `_`, which are only processed when passed
  as path themselves. A `/...` path also skips `testdata`, `vendor` and
  `node_modules` directories, directories starting with `.` or `_`, directories
  the module's `go.mod` ignores, and nested modules, so `-validate` no longer
  checks them. Unlike the `go` command, a `/...` path on a directory that is not
  within a module processes all modules below it, `go.work` is not read.
- `go-errs-wrap` keeps the import groups of the files it rewrites and formats them
  like `gofmt`; imports are only grouped with `goimports` when `replace` or `insert`
  adds the `go-errs` import. `remove -out` no longer runs `goimports`, it only
  removes the `go-errs` import when nothing uses it anymore.
- `go-errs-wrap -out` with a source directory without `/...` only processes and
  copies that directory, like without `-out`. Append `/...` to the path
  (`-out ./output ./pkg/...`) for the previous recursive output.
- `go-errs-wrap -out` with a source directory copies all files, not only the
  non-Go files and the rewritten ones, except version control metadata, symbolic
  links to directories and special files. It refuses an output path that is the
  source directory, is within it or contains it, also when spelled differently,
  because the output overwrote source files. All output stays within the output
  path: a symbolic link to a directory in it that leads outside is an error.
  Output files replace existing files and links instead of writing through them,
  and are writable by their owner, so read-only sources can be written to the
  output again. A single file is written with the permissions of its source.
- `go-errs-wrap -out ""`, like an unset variable in a script, is an error instead
  of modifying the source files in place.
- Requires Go 1.26 instead of Go 1.24, for all modules.

### Fixed

- `go-errs-wrap` compiles with go-astvisit `v0.0.0-20261013080339-68c5a526b805`,
  which `gen-func-wrappers` of go-function v1.0.0 requires: modules using both tools
  select that version for `go-errs-wrap` too, whose old import API it no longer has.
- `LogFunctionCall` no longer reads a `%` in the formatted call as format verb.
- `go-errs-wrap remove` without `-out` removes the `go-errs` import when nothing
  uses it anymore, so the rewritten file compiles.
- `go-errs-wrap` uses the name of an aliased `go-errs` import in generated
  statements instead of adding a second, unnamed import that left the alias unused.
- Where a receiver, type parameter, parameter or local declaration shadows the
  name of the `go-errs` import, like a parameter `errs ...error`, `go-errs-wrap`
  imports `go-errs` once more under a name that is free there and not used by
  another import (unnamed, or `goerrs`, `goerrs2` and so on) instead of
  generating a call on the shadowing variable. A method
  call on such a variable, like `defer e.WrapUp()` with a parameter `e`, is no
  longer taken for a `go-errs` statement.
- `go-errs-wrap -out` writes its output only after all files were rewritten,
  and keeps the CRLF line endings of rewritten files.
- `go-errs-wrap` fails for an argument after the path instead of ignoring it, so
  `go-errs-wrap insert ./... -validate` no longer rewrites files and exits with 0.
- `go-errs-wrap` imports `go-errs` as `goerrs` into a file that imports another
  package as `errs`, and no longer takes that package's `defer errs.Wrap*`
  statements for `go-errs` ones.
- The `go-errs-wrap` documentation lists exit code 2 for invalid options.
- `go-errs-wrap` processes a path ending in `\...` recursively on Windows,
  like the `go` command.

## [v1.0.4] - 2026-07-02

### Added

- Sentry stack trace support: errors wrapped with a call stack now expose a
  `StackTrace() []uintptr` method (the `pkg/errors` shape), so `sentry-go`'s
  `ExtractStacktrace` discovers their call stack via reflection with no import
  of, or dependency on, `sentry-go` or `pkg/errors`. The method is promoted
  through the embedded `withCallStack`, so both wrapper types report a stack
  trace to Sentry.

### Fixed

- Call-stack file paths are now rendered in a checkout-independent import-path
  form (e.g. `github.com/domonda/go-errs/format.go`), reconstructed from each
  frame's package instead of matching the literal string `github.com` in the
  build path. Previously, a module checked out under a path that did not
  contain `github.com` leaked absolute build paths into error messages.
  `TrimFilePathPrefix` now defaults to `""` and, when set, still trims the raw
  runtime path as before.

## [v1.0.3] - 2026-04-16

### Fixed

- `go-errs-wrap`: emit zero-parameter variadic wrap without a trailing comma.

## [v1.0.2] - 2026-04-16

### Fixed

- `go-errs-wrap`: skip no-op replacements to avoid import reordering.

## [v1.0.1] - 2026-03-08

### Fixed

- Add `nosec G703` annotations to suppress path-traversal false positives.
- Apply `go fix ./...`.

## [v1.0.0] - 2026-02-18

Initial stable release.

[v1.0.4]: https://github.com/domonda/go-errs/compare/v1.0.3...v1.0.4
[v1.0.3]: https://github.com/domonda/go-errs/compare/v1.0.2...v1.0.3
[v1.0.2]: https://github.com/domonda/go-errs/compare/v1.0.1...v1.0.2
[v1.0.1]: https://github.com/domonda/go-errs/compare/v1.0.0...v1.0.1
[v1.0.0]: https://github.com/domonda/go-errs/releases/tag/v1.0.0
