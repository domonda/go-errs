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
	_ = sentinel // if use is commented out for debugging

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

func TestType(t *testing.T) {
	sentinel := Sentinel("sentinel")

	tests := []struct {
		name string
		got  bool
		want bool
	}{
		{name: "nil", got: Type[Sentinel](nil), want: false},
		{name: "nil, error", got: Type[error](nil), want: false},
		{name: "sentinel, Sentinel", got: Type[Sentinel](sentinel), want: true},
		{name: "other, Sentinel", got: Type[Sentinel](errors.New("other")), want: false},
		{name: "struct, Sentinel", got: Type[Sentinel](errStruct{"other"}), want: false},
		{name: "struct, struct", got: Type[errStruct](errStruct{"a"}), want: true},
		{name: "wrapped(struct), struct", got: Type[errStruct](fmt.Errorf("wrapped: %w", errStruct{"a"})), want: true},
		{name: "2x wrapped(struct), struct", got: Type[errStruct](Errorf("wrapped: %w", errWrapper{Wrapped: errStruct{"a"}})), want: true},
		{name: "errWrapper(struct), errWrapper", got: Type[errWrapper](Errorf("wrapped: %w", errWrapper{Wrapped: errStruct{"a"}})), want: true},
		{name: "Errorf, withCallStack", got: Type[*withCallStack](Errorf("wrapped: %w", errWrapper{Wrapped: errStruct{"a"}})), want: true},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("Test %q: Type() = %v, want %v", tt.name, tt.got, tt.want)
		}
	}
}
