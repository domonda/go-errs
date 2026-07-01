# go-errs documentation

Structured documentation for `github.com/domonda/go-errs`, organized with the
[Diataxis](https://diataxis.fr) framework. Each section serves a different need:
learn the basics, get a specific task done, look up an exact detail, or
understand why the library works the way it does.

New here? Start with the [Getting started tutorial](tutorials/getting-started.md).
The project [README](../README.md) has the feature overview and quick start.

## Tutorials — learning-oriented

Start-to-finish walk-throughs for newcomers.

- [Getting started](tutorials/getting-started.md) — from install to a real
  multi-frame error trace, plus redacting a secret.

## How-to guides — task-oriented

Recipes for a specific goal, assuming you know the basics.

- [Wrap errors with function parameters](how-to/wrap-errors-with-function-parameters.md)
- [Redact sensitive parameters](how-to/redact-sensitive-parameters.md)
- [Handle not-found and context errors](how-to/handle-not-found-and-context-errors.md)
- [Recover panics as errors](how-to/recover-panics-as-errors.md)
- [Send stack traces to Sentry](how-to/send-stack-traces-to-sentry.md)
- [Manage wrapping with go-errs-wrap](how-to/manage-wrapping-with-go-errs-wrap.md)

## Reference — information-oriented

Complete, factual descriptions of the surface.

- [Package API](reference/api.md) — every exported function, type, and constant
- [Configuration](reference/configuration.md) — tunable package variables
- [go-errs-wrap CLI](reference/go-errs-wrap.md) — commands, flags, exit codes

## Explanation — understanding-oriented

The design rationale and trade-offs.

- [Call stacks and wrapper types](explanation/call-stacks-and-wrapper-types.md)
- [Sentry stack-trace interop](explanation/sentry-stack-trace-interop.md)
- [Secret redaction and pretty-printing](explanation/secret-redaction-and-pretty-printing.md)

## Finding what you need

| You want to…                              | Go to                                  |
| ----------------------------------------- | -------------------------------------- |
| Learn the library from scratch            | [Tutorial](tutorials/getting-started.md) |
| Accomplish a specific task                | [How-to guides](#how-to-guides--task-oriented) |
| Look up a function or option              | [Reference](#reference--information-oriented) |
| Understand a design decision              | [Explanation](#explanation--understanding-oriented) |
