package errs

import (
	"errors"
	"fmt"
	"testing"
)

type errStruct struct{ Err string }

func (e errStruct) Error() string { return e.Err }

type errWrapper struct{ Wrapped error }

func (e errWrapper) Error() string { return e.Wrapped.Error() }
func (e errWrapper) Unwrap() error { return e.Wrapped }

func TestRoot(t *testing.T) {
	sentinel := errors.New("sentinel")

	tests := []struct {
		name string
		err  error
		want error
	}{
		{name: "nil", err: nil, want: nil},
		{name: "not wrapped", err: sentinel, want: sentinel},
		{name: "1x wrapped", err: Errorf("wrapped: %w", sentinel), want: sentinel},
		{name: "1x std wrapped", err: fmt.Errorf("wrapped: %w", sentinel), want: sentinel},
		{name: "2x wrapped", err: Errorf("re-wrapped: %w", Errorf("wrapped: %w", sentinel)), want: sentinel},
		{name: "2x std wrapped", err: fmt.Errorf("re-wrapped: %w", fmt.Errorf("wrapped: %w", sentinel)), want: sentinel},
		{name: "2x mixed wrapped", err: fmt.Errorf("re-wrapped: %w", Errorf("wrapped: %w", sentinel)), want: sentinel},
		{name: "custom wrapped", err: errWrapper{Wrapped: sentinel}, want: sentinel},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Root(tt.err); err != tt.want {
				t.Errorf("Root() error = %v, want %v", err, tt.want)
			}
		})
	}
}

func TestIsType(t *testing.T) {
	sentinel := Sentinel("sentinel")

	// errors.As also works with the private error type of package errors
	target := errors.New("target")
	if !errors.As(errors.New("sentinel"), &target) || target.Error() != "sentinel" {
		t.Fatal("assumption of how errors.As works wrong")
	}

	type args struct {
		err    error
		target error
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{name: "nil", args: args{err: nil, target: sentinel}, want: false},
		{name: "nil, nil", args: args{err: nil, target: nil}, want: true},
		{name: "sentinel, sentinel", args: args{err: sentinel, target: sentinel}, want: true},
		{name: "other, sentinel", args: args{err: errors.New("other"), target: sentinel}, want: false},
		{name: "struct, sentinel", args: args{err: errStruct{"other"}, target: sentinel}, want: false},
		{name: "struct, struct", args: args{err: errStruct{"a"}, target: errStruct{"b"}}, want: true},
		{name: "wrapped(struct), struct", args: args{err: fmt.Errorf("wrapped: %w", errStruct{"a"}), target: errStruct{"b"}}, want: true},
		{name: "2x wrapped(struct), struct", args: args{err: Errorf("wrapped: %w", errWrapper{Wrapped: errStruct{"a"}}), target: errStruct{"b"}}, want: true},
		{name: "errWrapper(struct), errWrapper", args: args{err: Errorf("wrapped: %w", errWrapper{Wrapped: errStruct{"a"}}), target: errWrapper{}}, want: true},
		{name: "Errorf, withCallStack", args: args{err: Errorf("wrapped: %w", errWrapper{Wrapped: errStruct{"a"}}), target: New("withCallStack")}, want: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsType(tt.args.err, tt.args.target); got != tt.want {
				t.Errorf("IsType() = %v, want %v", got, tt.want)
			}
		})
	}
}
