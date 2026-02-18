package errs

import "fmt"

// Secret is an interface that wraps a secret value
// to prevent it from being logged or printed.
// It implements pretty.Stringer to ensure secrets
// are never revealed in error call stacks.
type Secret interface {
	// Secret returns the wrapped secret value.
	Secret() any

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

func (s secret) Secret() any {
	return s.val
}

func (secret) String() string {
	return "***REDACTED***"
}

func (s secret) GoString() string {
	return fmt.Sprintf("%T(***REDACTED***)", s.val)
}

// PrettyString implements the pretty.Stringer interface
// to ensure secrets are never revealed in pretty-printed output or error messages.
func (secret) PrettyString() string {
	return "***REDACTED***"
}
