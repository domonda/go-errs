# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [v1.0.4] - unreleased

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

[v1.0.4]: https://github.com/domonda/go-errs/compare/v1.0.3...HEAD
[v1.0.3]: https://github.com/domonda/go-errs/compare/v1.0.2...v1.0.3
[v1.0.2]: https://github.com/domonda/go-errs/compare/v1.0.1...v1.0.2
[v1.0.1]: https://github.com/domonda/go-errs/compare/v1.0.0...v1.0.1
[v1.0.0]: https://github.com/domonda/go-errs/releases/tag/v1.0.0
