# Getting started with go-errs

In this tutorial you will build a tiny program whose errors print *where* they
came from and *with what arguments* — the two things a plain Go error never
tells you. By the end you will understand call-stack capture, function-parameter
wrapping, and how they stack across calls.

You will see a real multi-frame error trace within the first three steps.

## What you'll need

- Go 1.24 or newer (`go version` to check).
- A scratch directory. No prior knowledge of the package required.

## Step 1: Create a module and install the package

```bash
mkdir errs-demo && cd errs-demo
go mod init errs-demo
go get github.com/domonda/go-errs
```

You now have a module with `go-errs` as a dependency.

## Step 2: Create an error with a call stack

Put this in `main.go`:

```go
package main

import (
    "fmt"

    "github.com/domonda/go-errs"
)

func main() {
    err := errs.New("something went wrong")
    fmt.Println(err)
}
```

Run it:

```bash
go run .
```

You will see the message *and* the line that produced it:

```
something went wrong
main.main
    /path/to/errs-demo/main.go:10
```

That is the whole idea: `errs.New` is a drop-in for `errors.New` that records
the call site for free. `errs.Errorf` does the same for `fmt.Errorf`, including
the `%w` wrapping verb.

## Step 3: Capture function parameters across calls

Replace `main.go` with three nested functions, each wrapping its own error:

```go
package main

import (
    "errors"
    "fmt"

    "github.com/domonda/go-errs"
)

func funcA(i int, s string) (err error) {
    defer errs.WrapWithFuncParams(&err, i, s)
    return funcB(s)
}

func funcB(s string) (err error) {
    defer errs.WrapWithFuncParams(&err, s)
    return funcC()
}

func funcC() (err error) {
    defer errs.WrapWithFuncParams(&err)
    return errors.New("error in funcC")
}

func main() {
    err := funcA(666, "Hello World!")
    fmt.Println(err.Error())
}
```

Run it again:

```bash
go run .
```

```
error in funcC
main.funcC()
    /path/to/errs-demo/main.go:22
main.funcB(`Hello World!`)
    /path/to/errs-demo/main.go:17
main.funcA(666, `Hello World!`)
    /path/to/errs-demo/main.go:12
```

Read it bottom to top and you have the whole story: `funcA` was called with
`666` and `"Hello World!"`, passed the string to `funcB`, which called `funcC`,
which failed. Each `defer errs.WrapWithFuncParams(&err, …)` added one line with
its own arguments. Three points to notice:

- The result must be a **named** error (`(err error)`) so the wrap can take
  `&err`.
- The wrap is a `defer`, so it runs on every return, but it is a **no-op when
  the error is nil** — safe to leave on.
- Strings render with backticks and slices as `[…]`, courtesy of the
  `go-pretty` formatter the package uses for parameters.

## Step 4: Protect a secret

Say `funcA`'s string was a password. Wrap it so it never reaches your logs:

```go
defer errs.WrapWithFuncParams(&err, i, errs.KeepSecret(s))
```

Re-run and the last frame now reads:

```
main.funcA(666, ***REDACTED***)
```

The value is still passed to your code; only its *rendering* is redacted.

## What you built

A program whose errors carry their origin call stack and argument values, with
one deferred line per function and a one-word change to hide a secret. That is
the core of `go-errs`. From here:

- Recipe-style guides: [How-to guides](../how-to/) — redaction, not-found and
  context errors, panic recovery, Sentry, and the CLI.
- The complete surface: [Package API reference](../reference/api.md).
- Why it works this way:
  [Call stacks and wrapper types](../explanation/call-stacks-and-wrapper-types.md).
- Stop writing the `defer` lines by hand: generate them across a package with
  [`go-errs-wrap`](../how-to/manage-wrapping-with-go-errs-wrap.md).
