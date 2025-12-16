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

func TestAs_WithErrorsJoin(t *testing.T) {
	t.Run("simple join", func(t *testing.T) {
		const (
			e0 = Sentinel("e0")
			e1 = Sentinel("e1")
			e2 = Sentinel("e2")
		)

		err := errors.Join(e0, e1, e2)
		errs := As[Sentinel](err)
		assert.Len(t, errs, 3)
		assert.Equal(t, e0, errs[0])
		assert.Equal(t, e1, errs[1])
		assert.Equal(t, e2, errs[2])
	})

	t.Run("mixed types in join", func(t *testing.T) {
		sentinel := Sentinel("sentinel")
		structErr := errStruct{Err: "struct"}
		wrappedErr := errWrapper{Wrapped: errors.New("wrapped")}

		err := errors.Join(sentinel, structErr, wrappedErr)

		sentinels := As[Sentinel](err)
		assert.Len(t, sentinels, 1)
		assert.Equal(t, sentinel, sentinels[0])

		structs := As[errStruct](err)
		assert.Len(t, structs, 1)
		assert.Equal(t, structErr, structs[0])

		wrappers := As[errWrapper](err)
		assert.Len(t, wrappers, 1)
		assert.Equal(t, wrappedErr, wrappers[0])
	})

	t.Run("nested join", func(t *testing.T) {
		e0 := Sentinel("e0")
		e1 := Sentinel("e1")
		e2 := Sentinel("e2")
		e3 := Sentinel("e3")

		inner := errors.Join(e1, e2)
		outer := errors.Join(e0, inner, e3)

		errs := As[Sentinel](outer)
		assert.Len(t, errs, 4)
		assert.Equal(t, e0, errs[0])
		assert.Equal(t, e1, errs[1])
		assert.Equal(t, e2, errs[2])
		assert.Equal(t, e3, errs[3])
	})

	t.Run("join with wrapped errors", func(t *testing.T) {
		e0 := Sentinel("e0")
		e1 := Sentinel("e1")
		wrapped := fmt.Errorf("wrapped: %w", e1)

		err := errors.Join(e0, wrapped)

		errs := As[Sentinel](err)
		assert.Len(t, errs, 2)
		assert.Equal(t, e0, errs[0])
		assert.Equal(t, e1, errs[1])
	})

	t.Run("join with callstack wrapped errors", func(t *testing.T) {
		e0 := Sentinel("e0")
		e1 := Sentinel("e1")
		withStack := WrapWithCallStack(e1)

		err := errors.Join(e0, withStack)

		errs := As[Sentinel](err)
		assert.Len(t, errs, 2)
		assert.Equal(t, e0, errs[0])
		assert.Equal(t, e1, errs[1])

		stacks := As[*withCallStack](err)
		assert.Len(t, stacks, 1)
	})
}

func TestUnwrapCallStack_WithErrorsJoin(t *testing.T) {
	t.Run("join with callstack wrapper", func(t *testing.T) {
		sentinel := Sentinel("sentinel")
		wrapped := WrapWithCallStack(sentinel)

		err := errors.Join(wrapped, errors.New("other"))

		// UnwrapCallStack should not affect errors.Join wrapper
		result := UnwrapCallStack(err)
		assert.Equal(t, err, result, "UnwrapCallStack should not unwrap errors.Join")
	})

	t.Run("callstack wrapped join", func(t *testing.T) {
		e0 := Sentinel("e0")
		e1 := Sentinel("e1")
		joined := errors.Join(e0, e1)
		wrapped := WrapWithCallStack(joined)

		result := UnwrapCallStack(wrapped)
		assert.Equal(t, joined, result)
	})

	t.Run("multiple callstack wrappers with join", func(t *testing.T) {
		sentinel := Sentinel("sentinel")
		wrapped1 := WrapWithCallStack(sentinel)
		wrapped2 := WrapWithCallStack(wrapped1)

		result := UnwrapCallStack(wrapped2)
		assert.Equal(t, sentinel, result)
	})
}

func TestHas_WithErrorsJoin(t *testing.T) {
	t.Run("find sentinel in join", func(t *testing.T) {
		sentinel := Sentinel("sentinel")
		other := errors.New("other")

		err := errors.Join(sentinel, other)

		assert.True(t, Has[Sentinel](err))
	})

	t.Run("find struct in join", func(t *testing.T) {
		structErr := errStruct{Err: "struct"}
		other := errors.New("other")

		err := errors.Join(structErr, other)

		assert.True(t, Has[errStruct](err))
		assert.False(t, Has[errWrapper](err))
	})

	t.Run("find in nested join", func(t *testing.T) {
		structErr := errStruct{Err: "struct"}
		inner := errors.Join(structErr, errors.New("inner"))
		outer := errors.Join(errors.New("outer"), inner)

		assert.True(t, Has[errStruct](outer))
	})
}

func TestType_WithErrorsJoin(t *testing.T) {
	t.Run("find type in join", func(t *testing.T) {
		sentinel := Sentinel("sentinel")
		other := errors.New("other")

		err := errors.Join(sentinel, other)

		assert.True(t, Type[Sentinel](err))
	})

	t.Run("find struct type in join", func(t *testing.T) {
		structErr := errStruct{Err: "struct"}
		other := errors.New("other")

		err := errors.Join(structErr, other)

		assert.True(t, Type[errStruct](err))
		assert.False(t, Type[errWrapper](err))
	})
}

func TestRoot_WithErrorsJoin(t *testing.T) {
	t.Run("root of errors.Join", func(t *testing.T) {
		e0 := Sentinel("e0")
		e1 := Sentinel("e1")

		err := errors.Join(e0, e1)

		// Root doesn't unwrap errors.Join since it doesn't implement single Unwrap()
		result := Root(err)
		assert.Equal(t, err, result)
	})

	t.Run("root through wrapped join", func(t *testing.T) {
		e0 := Sentinel("e0")
		e1 := Sentinel("e1")

		joined := errors.Join(e0, e1)
		wrapped := fmt.Errorf("wrapper: %w", joined)

		result := Root(wrapped)
		assert.Equal(t, joined, result)
	})
}

func TestUnwrapCallStack(t *testing.T) {
	sentinel := Sentinel("sentinel error")

	t.Run("nil error", func(t *testing.T) {
		result := UnwrapCallStack(nil)
		assert.Nil(t, result)
	})

	t.Run("error without callstack", func(t *testing.T) {
		err := errors.New("plain error")
		result := UnwrapCallStack(err)
		assert.Equal(t, err, result, "should return the same error if not wrapped with callstack")
	})

	t.Run("error with single callstack wrapper", func(t *testing.T) {
		wrapped := WrapWithCallStack(sentinel)
		result := UnwrapCallStack(wrapped)
		assert.Equal(t, sentinel, result, "should unwrap single callstack wrapper")
	})

	t.Run("error with multiple callstack wrappers", func(t *testing.T) {
		wrapped1 := WrapWithCallStack(sentinel)
		wrapped2 := WrapWithCallStack(wrapped1)
		wrapped3 := WrapWithCallStack(wrapped2)

		result := UnwrapCallStack(wrapped3)
		assert.Equal(t, sentinel, result, "should unwrap all callstack wrappers")
	})

	t.Run("error with callstack and func params", func(t *testing.T) {
		wrapped := wrapWithFuncParamsSkip(0, sentinel, "param1", 42)
		result := UnwrapCallStack(wrapped)
		assert.Equal(t, sentinel, result, "should unwrap callstack with func params")
	})

	t.Run("preserves error chain", func(t *testing.T) {
		// Create a chain: sentinel <- fmt.Errorf <- WrapWithCallStack
		wrapped1 := fmt.Errorf("wrapped: %w", sentinel)
		wrapped2 := WrapWithCallStack(wrapped1)

		result := UnwrapCallStack(wrapped2)
		assert.Equal(t, wrapped1, result, "should only remove top callstack, preserving error chain")

		// The result should still be able to unwrap to sentinel
		assert.ErrorIs(t, result, sentinel)
	})

	t.Run("does not unwrap non-callstack wrappers", func(t *testing.T) {
		// Create: sentinel <- fmt.Errorf
		wrapped := fmt.Errorf("context: %w", sentinel)

		result := UnwrapCallStack(wrapped)
		assert.Equal(t, wrapped, result, "should not unwrap non-callstack errors")
	})

	t.Run("callstack in middle of chain", func(t *testing.T) {
		// Create: sentinel <- WrapWithCallStack <- fmt.Errorf <- WrapWithCallStack
		wrapped1 := WrapWithCallStack(sentinel)
		wrapped2 := fmt.Errorf("context: %w", wrapped1)
		wrapped3 := WrapWithCallStack(wrapped2)

		result := UnwrapCallStack(wrapped3)
		assert.Equal(t, wrapped2, result, "should only remove top-level callstack")

		// The wrapped1 callstack should still be in the chain
		assert.True(t, Has[*withCallStack](result), "callstack wrapper in middle should be preserved")
	})

	t.Run("difference between UnwrapCallStack and Root", func(t *testing.T) {
		// Create: sentinel <- fmt.Errorf <- WrapWithCallStack
		wrapped1 := fmt.Errorf("layer1: %w", sentinel)
		wrapped2 := WrapWithCallStack(wrapped1)

		// UnwrapCallStack only removes callstack wrappers
		withoutStack := UnwrapCallStack(wrapped2)
		assert.Equal(t, wrapped1, withoutStack)

		// Root unwraps everything to the root cause
		root := Root(wrapped2)
		assert.Equal(t, sentinel, root)

		assert.NotEqual(t, withoutStack, root, "UnwrapCallStack and Root should produce different results")
	})

	t.Run("with errors.Join", func(t *testing.T) {
		e0 := Sentinel("e0")
		e1 := Sentinel("e1")
		joined := errors.Join(e0, e1)
		wrapped := WrapWithCallStack(joined)

		result := UnwrapCallStack(wrapped)
		assert.Equal(t, joined, result, "should unwrap callstack from errors.Join")
	})

	t.Run("custom wrapper type", func(t *testing.T) {
		custom := errWrapper{Wrapped: sentinel}
		wrapped := WrapWithCallStack(custom)

		result := UnwrapCallStack(wrapped)
		assert.Equal(t, custom, result, "should preserve custom wrapper types")
	})

	t.Run("comparison use case", func(t *testing.T) {
		// Demonstrate using UnwrapCallStack for error comparison
		err1 := WrapWithCallStack(sentinel)
		err2 := WrapWithCallStack(sentinel)

		// Different callstack wrappers are not equal
		assert.NotEqual(t, err1, err2, "errors with different callstacks should not be equal")

		// But unwrapping reveals the same underlying error
		unwrapped1 := UnwrapCallStack(err1)
		unwrapped2 := UnwrapCallStack(err2)
		assert.Equal(t, unwrapped1, unwrapped2, "unwrapped errors should be equal")
		assert.Equal(t, sentinel, unwrapped1)
		assert.Equal(t, sentinel, unwrapped2)
	})

	t.Run("mixed callstack types", func(t *testing.T) {
		// Mix WrapWithCallStack and wrapWithFuncParamsSkip
		wrapped1 := WrapWithCallStack(sentinel)
		wrapped2 := wrapWithFuncParamsSkip(0, wrapped1, "param")
		wrapped3 := WrapWithCallStack(wrapped2)

		result := UnwrapCallStack(wrapped3)
		assert.Equal(t, sentinel, result, "should unwrap all types of callstack wrappers")
	})
}
