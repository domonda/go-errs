package errs

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
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

func TestHas(t *testing.T) {
	sentinel := Sentinel("sentinel")

	tests := []struct {
		name string
		got  bool
		want bool
	}{
		{name: "nil", got: Has[Sentinel](nil), want: false},
		{name: "nil, error", got: Has[error](nil), want: false},
		{name: "sentinel, Sentinel", got: Has[Sentinel](sentinel), want: true},
		{name: "other, Sentinel", got: Has[Sentinel](errors.New("other")), want: false},
		{name: "struct, Sentinel", got: Has[Sentinel](errStruct{"other"}), want: false},
		{name: "struct, struct", got: Has[errStruct](errStruct{"a"}), want: true},
		{name: "wrapped(struct), struct", got: Has[errStruct](fmt.Errorf("wrapped: %w", errStruct{"a"})), want: true},
		{name: "2x wrapped(struct), struct", got: Has[errStruct](Errorf("wrapped: %w", errWrapper{Wrapped: errStruct{"a"}})), want: true},
		{name: "errWrapper(struct), errWrapper", got: Has[errWrapper](Errorf("wrapped: %w", errWrapper{Wrapped: errStruct{"a"}})), want: true},
		{name: "Errorf, withCallStack", got: Has[*withCallStack](Errorf("wrapped: %w", errWrapper{Wrapped: errStruct{"a"}})), want: true},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("Test %q: Type() = %v, want %v", tt.name, tt.got, tt.want)
		}
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

// func Test_UnwrapAll(t *testing.T) {
// 	const (
// 		e0 = Sentinel("e0")
// 		e1 = Sentinel("e1")
// 		e2 = Sentinel("e2")
// 	)

// 	err := errors.Join(e0, e1, e2)
// 	assert.EqualError(t, err, "e0\ne1\ne2")

// 	errs := UnwrapAll(err)
// 	assert.Len(t, errs, 3)
// 	assert.Equal(t, e0, errs[0])
// 	assert.Equal(t, e1, errs[1])
// 	assert.Equal(t, e2, errs[2])
// }

func Test_As(t *testing.T) {
	const (
		e0 = Sentinel("e0")
		e1 = Sentinel("e1")
		e2 = Sentinel("e2")
	)

	err := errors.Join(e0, e1, e2)
	assert.EqualError(t, err, "e0\ne1\ne2")

	errs := As[Sentinel](err)
	assert.Len(t, errs, 3)
	assert.Equal(t, e0, errs[0])
	assert.Equal(t, e1, errs[1])
	assert.Equal(t, e2, errs[2])
}
