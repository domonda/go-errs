package errs

import (
	"fmt"
	"io"
)

// Secret is an interface that wraps a secret value
// to prevent it from being logged or printed.
// It implements CallStackPrintable to ensure secrets
// are never revealed in error call stacks.
type Secret interface {
	// Secrect returns the wrapped secret value.
	Secrect() any

	// String returns a redacted string that indicates
	// that the value is a secret without revealing the actual value.
	String() string
}

// KeepSecret wraps the passed value in a Secret
// to prevent it from being logged or printed.
func KeepSecret(val any) Secret {
	return secret{val}
}

type secret struct{ val any }

func (s secret) Secrect() any {
	return s.val
}

func (secret) String() string {
	return "***REDACTED***"
}

func (s secret) GoString() string {
	return fmt.Sprintf("%T(***REDACTED***)", s.val)
}

// PrettyPrint implements the pretty.Printable interface
// to ensure secrets are never revealed in pretty-printed output or error messages.
func (secret) PrettyPrint(w io.Writer) {
	io.WriteString(w, "***REDACTED***") // #nosec G104 -- intentionally ignoring error for simple string write
}
