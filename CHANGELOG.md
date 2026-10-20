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
- `go-errs-wrap` selects the files of a `/...` path like the `go` command:
  `main` packages are now processed, while `testdata` and `vendor` directories,
  directories starting with `.` or `_`, nested modules and files that need
  custom build tags are skipped, so `-validate` no longer checks them.
- `go-errs-wrap` keeps the import groups of a file that already imports `go-errs`
  and formats it like `gofmt`; imports are only grouped with `goimports`
  when the `go-errs` import is added.
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
- `go-errs-wrap -out` with a source directory also copies the Go files it does
  not change, and like without `-out` only processes subdirectories for a `/...` path.

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
