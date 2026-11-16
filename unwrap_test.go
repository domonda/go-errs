package errs

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		{name: "custom wrapper of nil", err: errWrapper{}, want: errWrapper{}},
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
		{name: "errWrapper(nil), struct", args: args{err: errWrapper{}, target: errStruct{"a"}}, want: false},
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

		// Root traverses into the join and returns the root of the first branch
		result := Root(err)
		assert.Equal(t, e0, result)
	})

	t.Run("root through wrapped join", func(t *testing.T) {
		e0 := Sentinel("e0")
		e1 := Sentinel("e1")

		joined := errors.Join(e0, e1)
		wrapped := fmt.Errorf("wrapper: %w", joined)

		// Root unwraps through the fmt.Errorf wrapper, into the join, and returns root of first branch
		result := Root(wrapped)
		assert.Equal(t, e0, result)
	})

	t.Run("root of deeply wrapped join", func(t *testing.T) {
		deep := Sentinel("deep")
		wrapped := fmt.Errorf("mid: %w", deep)
		other := Sentinel("other")

		joined := errors.Join(wrapped, other)

		// Root follows first branch (wrapped) and unwraps to deep
		result := Root(joined)
		assert.Equal(t, deep, result)
	})

	t.Run("nil error", func(t *testing.T) {
		result := Root(nil)
		assert.Nil(t, result)
	})

	t.Run("multi-error with only nil errors is the root", func(t *testing.T) {
		err := multiWrapper{errs: []error{nil, nil}}

		assert.Equal(t, err, Root(err))
	})
}

func TestIsType_WithErrorsJoin(t *testing.T) {
	t.Run("find type in join", func(t *testing.T) {
		structErr := errStruct{Err: "found"}
		other := errors.New("other")

		joined := errors.Join(other, structErr)

		assert.True(t, IsType(joined, errStruct{}))
	})

	t.Run("find type in nested join", func(t *testing.T) {
		structErr := errStruct{Err: "deep"}
		wrapped := fmt.Errorf("wrap: %w", structErr)
		other := errors.New("other")

		joined := errors.Join(other, wrapped)

		assert.True(t, IsType(joined, errStruct{}))
	})

	t.Run("type not in join", func(t *testing.T) {
		e1 := errors.New("e1")
		e2 := Sentinel("e2")

		joined := errors.Join(e1, e2)

		assert.False(t, IsType(joined, errStruct{}))
	})
}

// Custom error types for comprehensive As testing

// customAsError implements As(any) bool to match specific types
type customAsError struct {
	msg       string
	matchType string // "sentinel", "errStruct", or "none"
}

func (e customAsError) Error() string { return e.msg }

func (e customAsError) As(target any) bool {
	switch e.matchType {
	case "sentinel":
		if p, ok := target.(*Sentinel); ok {
			*p = Sentinel("matched-via-as:" + e.msg)
			return true
		}
	case "errStruct":
		if p, ok := target.(*errStruct); ok {
			*p = errStruct{Err: "matched-via-as:" + e.msg}
			return true
		}
	}
	return false
}

// wrappingAsError implements both Unwrap() error and As(any) bool
type wrappingAsError struct {
	wrapped   error
	matchType string
}

func (e wrappingAsError) Error() string { return "wrappingAs: " + e.wrapped.Error() }
func (e wrappingAsError) Unwrap() error { return e.wrapped }

func (e wrappingAsError) As(target any) bool {
	if e.matchType == "sentinel" {
		if p, ok := target.(*Sentinel); ok {
			*p = Sentinel("wrapping-as-matched")
			return true
		}
	}
	return false
}

// multiWrapper implements Unwrap() []error
type multiWrapper struct {
	errs []error
}

func (e multiWrapper) Error() string {
	var msg strings.Builder
	msg.WriteString("multi:")
	for i, err := range e.errs {
		if i > 0 {
			msg.WriteString(",")
		}
		msg.WriteString(err.Error())
	}
	return msg.String()
}

func (e multiWrapper) Unwrap() []error { return e.errs }

// multiWrapperWithAs implements both Unwrap() []error and As(any) bool
type multiWrapperWithAs struct {
	errs      []error
	matchType string
}

func (e multiWrapperWithAs) Error() string   { return "multiWithAs" }
func (e multiWrapperWithAs) Unwrap() []error { return e.errs }

func (e multiWrapperWithAs) As(target any) bool {
	if e.matchType == "sentinel" {
		if p, ok := target.(*Sentinel); ok {
			*p = Sentinel("multi-wrapper-as-matched")
			return true
		}
	}
	return false
}

// deepWrapper creates a chain of single-wrapped errors
func deepWrapper(depth int, root error) error {
	err := root
	for range depth {
		err = errWrapper{Wrapped: err}
	}
	return err
}

func TestAs_Comprehensive(t *testing.T) {
	t.Run("nil error", func(t *testing.T) {
		result := As[Sentinel](nil)
		assert.Nil(t, result)
		assert.Len(t, result, 0)
	})

	t.Run("direct type match", func(t *testing.T) {
		sentinel := Sentinel("direct")
		result := As[Sentinel](sentinel)
		assert.Len(t, result, 1)
		assert.Equal(t, sentinel, result[0])
	})

	t.Run("no match", func(t *testing.T) {
		err := errors.New("standard error")
		result := As[Sentinel](err)
		assert.Len(t, result, 0)
	})
}

func TestAs_UnwrapSingleError(t *testing.T) {
	t.Run("single level wrap", func(t *testing.T) {
		sentinel := Sentinel("wrapped")
		wrapped := errWrapper{Wrapped: sentinel}

		result := As[Sentinel](wrapped)
		assert.Len(t, result, 1)
		assert.Equal(t, sentinel, result[0])
	})

	t.Run("deep single-error chain", func(t *testing.T) {
		sentinel := Sentinel("deep")
		wrapped := deepWrapper(10, sentinel)

		result := As[Sentinel](wrapped)
		assert.Len(t, result, 1)
		assert.Equal(t, sentinel, result[0])
	})

	t.Run("multiple matches in single chain", func(t *testing.T) {
		inner := errStruct{Err: "inner"}
		middle := errWrapper{Wrapped: inner}
		outer := errWrapper{Wrapped: middle}

		// Should find both errWrapper instances
		result := As[errWrapper](outer)
		assert.Len(t, result, 2)
		assert.Equal(t, outer, result[0])
		assert.Equal(t, middle, result[1])
	})

	t.Run("very deep chain with multiple matches", func(t *testing.T) {
		e1 := errStruct{Err: "e1"}
		e2 := errStruct{Err: "e2"}
		e3 := errStruct{Err: "e3"}

		// Create: e3 <- wrapper <- wrapper <- e2 <- wrapper <- e1
		chain := errWrapper{Wrapped: e1}
		chain = errWrapper{Wrapped: errWrapper{Wrapped: e2}}
		chain = errWrapper{Wrapped: errWrapper{Wrapped: errWrapper{Wrapped: e3}}}

		// The chain is now: wrapper -> wrapper -> wrapper -> e3
		result := As[errStruct](chain)
		assert.Len(t, result, 1)
		assert.Equal(t, e3, result[0])
	})

	t.Run("fmt.Errorf wrapped errors", func(t *testing.T) {
		sentinel := Sentinel("fmt-wrapped")
		wrapped := fmt.Errorf("context: %w", sentinel)

		result := As[Sentinel](wrapped)
		assert.Len(t, result, 1)
		assert.Equal(t, sentinel, result[0])
	})

	t.Run("deeply nested fmt.Errorf", func(t *testing.T) {
		sentinel := Sentinel("deep-fmt")
		wrapped := fmt.Errorf("l1: %w", fmt.Errorf("l2: %w", fmt.Errorf("l3: %w", sentinel)))

		result := As[Sentinel](wrapped)
		assert.Len(t, result, 1)
		assert.Equal(t, sentinel, result[0])
	})
}

func TestAs_UnwrapMultipleErrors(t *testing.T) {
	t.Run("simple multi-wrapper", func(t *testing.T) {
		e1 := Sentinel("e1")
		e2 := Sentinel("e2")
		e3 := Sentinel("e3")

		multi := multiWrapper{errs: []error{e1, e2, e3}}

		result := As[Sentinel](multi)
		assert.Len(t, result, 3)
		assert.Equal(t, e1, result[0])
		assert.Equal(t, e2, result[1])
		assert.Equal(t, e3, result[2])
	})

	t.Run("multi-wrapper with nil errors", func(t *testing.T) {
		e1 := Sentinel("e1")
		e2 := Sentinel("e2")

		multi := multiWrapper{errs: []error{e1, nil, e2, nil}}

		result := As[Sentinel](multi)
		assert.Len(t, result, 2)
		assert.Equal(t, e1, result[0])
		assert.Equal(t, e2, result[1])
	})

	t.Run("nested multi-wrappers", func(t *testing.T) {
		e1 := Sentinel("e1")
		e2 := Sentinel("e2")
		e3 := Sentinel("e3")
		e4 := Sentinel("e4")

		inner := multiWrapper{errs: []error{e2, e3}}
		outer := multiWrapper{errs: []error{e1, inner, e4}}

		result := As[Sentinel](outer)
		assert.Len(t, result, 4)
		assert.Equal(t, e1, result[0])
		assert.Equal(t, e2, result[1])
		assert.Equal(t, e3, result[2])
		assert.Equal(t, e4, result[3])
	})

	t.Run("multi-wrapper mixed with single-wrapper", func(t *testing.T) {
		e1 := Sentinel("e1")
		e2 := Sentinel("e2")
		e3 := Sentinel("e3")

		// e2 wrapped in single wrapper
		singleWrapped := errWrapper{Wrapped: e2}
		multi := multiWrapper{errs: []error{e1, singleWrapped, e3}}

		result := As[Sentinel](multi)
		assert.Len(t, result, 3)
		assert.Equal(t, e1, result[0])
		assert.Equal(t, e2, result[1])
		assert.Equal(t, e3, result[2])
	})

	t.Run("deeply nested tree", func(t *testing.T) {
		// Create a complex tree:
		//         root (multi)
		//        /     \
		//    branch1   branch2 (multi)
		//      |       /    \
		//    leaf1  leaf2  leaf3

		leaf1 := Sentinel("leaf1")
		leaf2 := Sentinel("leaf2")
		leaf3 := Sentinel("leaf3")

		branch1 := errWrapper{Wrapped: leaf1}
		branch2 := multiWrapper{errs: []error{leaf2, leaf3}}
		root := multiWrapper{errs: []error{branch1, branch2}}

		result := As[Sentinel](root)
		assert.Len(t, result, 3)
		assert.Equal(t, leaf1, result[0])
		assert.Equal(t, leaf2, result[1])
		assert.Equal(t, leaf3, result[2])
	})

	t.Run("errors.Join behavior", func(t *testing.T) {
		e1 := Sentinel("joined1")
		e2 := Sentinel("joined2")
		e3 := Sentinel("joined3")

		joined := errors.Join(e1, e2, e3)

		result := As[Sentinel](joined)
		assert.Len(t, result, 3)
	})

	t.Run("mixed types in multi-wrapper", func(t *testing.T) {
		sentinel := Sentinel("sentinel")
		structErr := errStruct{Err: "struct"}
		plainErr := errors.New("plain")

		multi := multiWrapper{errs: []error{sentinel, structErr, plainErr}}

		sentinels := As[Sentinel](multi)
		assert.Len(t, sentinels, 1)
		assert.Equal(t, sentinel, sentinels[0])

		structs := As[errStruct](multi)
		assert.Len(t, structs, 1)
		assert.Equal(t, structErr, structs[0])
	})
}

func TestAs_CustomAsMethod(t *testing.T) {
	t.Run("direct As match - sentinel", func(t *testing.T) {
		customErr := customAsError{msg: "custom", matchType: "sentinel"}

		result := As[Sentinel](customErr)
		assert.Len(t, result, 1)
		assert.Equal(t, Sentinel("matched-via-as:custom"), result[0])
	})

	t.Run("direct As match - errStruct", func(t *testing.T) {
		customErr := customAsError{msg: "custom", matchType: "errStruct"}

		result := As[errStruct](customErr)
		assert.Len(t, result, 1)
		assert.Equal(t, errStruct{Err: "matched-via-as:custom"}, result[0])
	})

	t.Run("As method returns false", func(t *testing.T) {
		customErr := customAsError{msg: "custom", matchType: "none"}

		result := As[Sentinel](customErr)
		assert.Len(t, result, 0)
	})

	t.Run("type assertion takes precedence over As method", func(t *testing.T) {
		// When the error directly matches the type, As method should NOT be called
		sentinel := Sentinel("direct-match")

		// Wrap it in something that also implements As for Sentinel
		wrapped := wrappingAsError{wrapped: sentinel, matchType: "sentinel"}

		result := As[Sentinel](wrapped)
		// Should find: wrapping-as-matched (from As method), then direct-match (from unwrap)
		// But the wrapper's As should be used because type assertion fails for wrappingAsError -> Sentinel
		assert.Len(t, result, 2)
		assert.Equal(t, Sentinel("wrapping-as-matched"), result[0])
		assert.Equal(t, sentinel, result[1])
	})

	t.Run("wrapped custom As error", func(t *testing.T) {
		customErr := customAsError{msg: "wrapped-custom", matchType: "sentinel"}
		wrapped := errWrapper{Wrapped: customErr}

		result := As[Sentinel](wrapped)
		assert.Len(t, result, 1)
		assert.Equal(t, Sentinel("matched-via-as:wrapped-custom"), result[0])
	})

	t.Run("custom As in multi-wrapper", func(t *testing.T) {
		e1 := Sentinel("direct1")
		customErr := customAsError{msg: "custom", matchType: "sentinel"}
		e2 := Sentinel("direct2")

		multi := multiWrapper{errs: []error{e1, customErr, e2}}

		result := As[Sentinel](multi)
		assert.Len(t, result, 3)
		assert.Equal(t, e1, result[0])
		assert.Equal(t, Sentinel("matched-via-as:custom"), result[1])
		assert.Equal(t, e2, result[2])
	})

	t.Run("multi-wrapper with As method", func(t *testing.T) {
		e1 := Sentinel("e1")
		e2 := Sentinel("e2")

		multi := multiWrapperWithAs{errs: []error{e1, e2}, matchType: "sentinel"}

		result := As[Sentinel](multi)
		// Should get: match from As method, then e1 and e2 from children
		assert.Len(t, result, 3)
		assert.Equal(t, Sentinel("multi-wrapper-as-matched"), result[0])
		assert.Equal(t, e1, result[1])
		assert.Equal(t, e2, result[2])
	})
}

func TestAs_ComplexTrees(t *testing.T) {
	t.Run("wide and deep tree", func(t *testing.T) {
		// Create a tree that is both wide and deep:
		//              root (multi)
		//     /          |           \
		//  branch1    branch2      branch3 (multi)
		//    |          |          /    \
		//  deep1      deep2     leaf4  leaf5
		//    |          |
		//  leaf1      leaf2,leaf3 (multi)

		leaf1 := Sentinel("leaf1")
		leaf2 := Sentinel("leaf2")
		leaf3 := Sentinel("leaf3")
		leaf4 := Sentinel("leaf4")
		leaf5 := Sentinel("leaf5")

		deep1 := errWrapper{Wrapped: leaf1}
		deep2 := multiWrapper{errs: []error{leaf2, leaf3}}

		branch1 := errWrapper{Wrapped: deep1}
		branch2 := errWrapper{Wrapped: deep2}
		branch3 := multiWrapper{errs: []error{leaf4, leaf5}}

		root := multiWrapper{errs: []error{branch1, branch2, branch3}}

		result := As[Sentinel](root)
		assert.Len(t, result, 5)
		assert.Equal(t, leaf1, result[0])
		assert.Equal(t, leaf2, result[1])
		assert.Equal(t, leaf3, result[2])
		assert.Equal(t, leaf4, result[3])
		assert.Equal(t, leaf5, result[4])
	})

	t.Run("tree with As methods at different levels", func(t *testing.T) {
		leaf := Sentinel("leaf")
		customDeep := customAsError{msg: "deep-custom", matchType: "sentinel"}

		inner := multiWrapper{errs: []error{leaf, customDeep}}
		outer := wrappingAsError{wrapped: inner, matchType: "sentinel"}

		result := As[Sentinel](outer)
		// Should get: wrapping-as-matched (outer As), leaf (direct), matched-via-as:deep-custom (inner As)
		assert.Len(t, result, 3)
		assert.Equal(t, Sentinel("wrapping-as-matched"), result[0])
		assert.Equal(t, leaf, result[1])
		assert.Equal(t, Sentinel("matched-via-as:deep-custom"), result[2])
	})

	t.Run("single-wrap chain ending in multi-wrap", func(t *testing.T) {
		e1 := Sentinel("e1")
		e2 := Sentinel("e2")

		multi := multiWrapper{errs: []error{e1, e2}}
		wrapped := errWrapper{Wrapped: errWrapper{Wrapped: errWrapper{Wrapped: multi}}}

		result := As[Sentinel](wrapped)
		assert.Len(t, result, 2)
		assert.Equal(t, e1, result[0])
		assert.Equal(t, e2, result[1])
	})

	t.Run("multi-wrap containing single-wrap chains", func(t *testing.T) {
		e1 := Sentinel("e1")
		e2 := Sentinel("e2")
		e3 := Sentinel("e3")

		chain1 := errWrapper{Wrapped: errWrapper{Wrapped: e1}}
		chain2 := errWrapper{Wrapped: e2}
		chain3 := errWrapper{Wrapped: errWrapper{Wrapped: errWrapper{Wrapped: e3}}}

		multi := multiWrapper{errs: []error{chain1, chain2, chain3}}

		result := As[Sentinel](multi)
		assert.Len(t, result, 3)
		assert.Equal(t, e1, result[0])
		assert.Equal(t, e2, result[1])
		assert.Equal(t, e3, result[2])
	})

	t.Run("alternating single and multi wrappers", func(t *testing.T) {
		leaf := Sentinel("leaf")

		// Create: single -> multi -> single -> multi -> leaf
		level4 := multiWrapper{errs: []error{leaf}}
		level3 := errWrapper{Wrapped: level4}
		level2 := multiWrapper{errs: []error{level3}}
		level1 := errWrapper{Wrapped: level2}

		result := As[Sentinel](level1)
		assert.Len(t, result, 1)
		assert.Equal(t, leaf, result[0])
	})

	t.Run("real-world scenario: multiple error sources", func(t *testing.T) {
		// Simulate a scenario where multiple operations failed
		dbErr := errStruct{Err: "database connection failed"}
		apiErr := errStruct{Err: "API timeout"}
		fileErr := errStruct{Err: "file not found"}

		// Each wrapped with context
		wrappedDB := fmt.Errorf("user lookup: %w", dbErr)
		wrappedAPI := fmt.Errorf("external service: %w", apiErr)
		wrappedFile := fmt.Errorf("config load: %w", fileErr)

		// Combined using errors.Join
		combined := errors.Join(wrappedDB, wrappedAPI, wrappedFile)

		result := As[errStruct](combined)
		assert.Len(t, result, 3)
		assert.Equal(t, dbErr, result[0])
		assert.Equal(t, apiErr, result[1])
		assert.Equal(t, fileErr, result[2])
	})

	t.Run("empty multi-wrapper", func(t *testing.T) {
		multi := multiWrapper{errs: []error{}}

		result := As[Sentinel](multi)
		assert.Len(t, result, 0)
	})

	t.Run("multi-wrapper with only nil errors", func(t *testing.T) {
		multi := multiWrapper{errs: []error{nil, nil, nil}}

		result := As[Sentinel](multi)
		assert.Len(t, result, 0)
	})
}

func TestAs_EdgeCases(t *testing.T) {
	t.Run("interface type matching", func(t *testing.T) {
		sentinel := Sentinel("sentinel")
		wrapped := errWrapper{Wrapped: sentinel}

		// Match against the error interface itself
		result := As[error](wrapped)
		assert.Len(t, result, 2)
	})

	t.Run("pointer vs value type", func(t *testing.T) {
		structErr := errStruct{Err: "struct"}
		wrapped := errWrapper{Wrapped: structErr}

		// Value type
		valueResult := As[errStruct](wrapped)
		assert.Len(t, valueResult, 1)

		// Pointer type should not match value
		ptrResult := As[*errStruct](wrapped)
		assert.Len(t, ptrResult, 0)
	})

	t.Run("self-referential error handling", func(t *testing.T) {
		// This shouldn't cause infinite loop due to the traversal structure
		sentinel := Sentinel("sentinel")
		result := As[Sentinel](sentinel)
		assert.Len(t, result, 1)
	})

	t.Run("deeply nested single chain performance", func(t *testing.T) {
		sentinel := Sentinel("deep")
		wrapped := deepWrapper(100, sentinel)

		result := As[Sentinel](wrapped)
		assert.Len(t, result, 1)
		assert.Equal(t, sentinel, result[0])
	})

	t.Run("wide tree performance", func(t *testing.T) {
		// Create a wide tree with many siblings
		errs := make([]error, 100)
		for i := range errs {
			errs[i] = Sentinel(fmt.Sprintf("sentinel-%d", i))
		}

		multi := multiWrapper{errs: errs}

		result := As[Sentinel](multi)
		assert.Len(t, result, 100)
	})

	t.Run("As method not called when type matches directly", func(t *testing.T) {
		// customAsError itself matches customAsError type, so As shouldn't be called
		customErr := customAsError{msg: "self", matchType: "sentinel"}

		result := As[customAsError](customErr)
		assert.Len(t, result, 1)
		// Should be the original, not transformed by As
		assert.Equal(t, customErr, result[0])
	})

	t.Run("multiple different types in same tree", func(t *testing.T) {
		sentinel1 := Sentinel("s1")
		sentinel2 := Sentinel("s2")
		struct1 := errStruct{Err: "st1"}
		struct2 := errStruct{Err: "st2"}
		wrapper1 := errWrapper{Wrapped: sentinel1}

		// Mix all types in a tree
		inner := multiWrapper{errs: []error{sentinel2, struct2}}
		outer := multiWrapper{errs: []error{wrapper1, struct1, inner}}

		sentinels := As[Sentinel](outer)
		assert.Len(t, sentinels, 2)

		structs := As[errStruct](outer)
		assert.Len(t, structs, 2)

		wrappers := As[errWrapper](outer)
		assert.Len(t, wrappers, 1)
	})
}

func TestAs_ComparisonWithStdlibErrorsAs(t *testing.T) {
	t.Run("stdlib As finds first match only", func(t *testing.T) {
		e1 := Sentinel("e1")
		e2 := Sentinel("e2")

		multi := multiWrapper{errs: []error{e1, e2}}

		// stdlib errors.As only finds the first one
		var target Sentinel
		found := errors.As(multi, &target)
		assert.True(t, found)
		assert.Equal(t, e1, target)

		// Our As finds all
		result := As[Sentinel](multi)
		assert.Len(t, result, 2)
	})

	t.Run("behavior matches stdlib for single chain", func(t *testing.T) {
		sentinel := Sentinel("match")
		wrapped := errWrapper{Wrapped: sentinel}

		// stdlib
		var target Sentinel
		found := errors.As(wrapped, &target)
		assert.True(t, found)
		assert.Equal(t, sentinel, target)

		// Our As
		result := As[Sentinel](wrapped)
		assert.Len(t, result, 1)
		assert.Equal(t, sentinel, result[0])
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

func TestAs_NilWrappedErrors(t *testing.T) {
	t.Run("wrapper of nil", func(t *testing.T) {
		assert.Empty(t, As[errStruct](errWrapper{}))
	})

	t.Run("multi-error skips nil errors", func(t *testing.T) {
		err := multiWrapper{errs: []error{nil, errStruct{Err: "a"}, nil}}

		assert.Equal(t, []errStruct{{Err: "a"}}, As[errStruct](err))
	})
}

func TestType(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "nil", err: nil, want: false},
		{name: "struct", err: errStruct{Err: "a"}, want: true},
		{name: "wrapped struct", err: fmt.Errorf("wrapped: %w", errStruct{Err: "a"}), want: true},
		{name: "2x wrapped struct", err: Errorf("wrapped: %w", errWrapper{Wrapped: errStruct{Err: "a"}}), want: true},
		{name: "other", err: errors.New("other"), want: false},
		{name: "wrapper of nil", err: errWrapper{}, want: false},
		// Type does not consult As methods, unlike errors.As
		{name: "As method", err: customAsError{msg: "as", matchType: "errStruct"}, want: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, Type[errStruct](tt.err))
		})
	}
}

// validateReadOnlyQuery returns an error wrapped with its call stack
// and the query as function parameter
func validateReadOnlyQuery(query string) (err error) {
	defer WrapWithFuncParams(&err, query)

	if strings.Contains(query, "set_config") {
		return New("query contains disallowed identifier: set_config")
	}
	return nil
}

func ExampleMessageWithoutCallStacks() {
	err := validateReadOnlyQuery("SELECT set_config('role', 'admin', false)")
	err = Errorf("invalid query: %w", err)

	// err.Error() contains both call stacks and the query for the logs,
	// the client gets only the message
	fmt.Println(strings.Contains(err.Error(), "unwrap_test.go:"))
	fmt.Println(MessageWithoutCallStacks(err))

	// Output:
	// true
	// invalid query: query contains disallowed identifier: set_config
}

// upperError reformats the message of its wrapped error
type upperError struct{ wrapped error }

func (e upperError) Error() string { return strings.ToUpper(e.wrapped.Error()) }
func (e upperError) Unwrap() error { return e.wrapped }

// ptrError is an error with a pointer receiver,
// its Error method panics for a nil pointer
type ptrError struct{ msg string }

func (e *ptrError) Error() string { return e.msg }

// panicAfterNilPointerResult panics after setting its result
// to a nil pointer error, the result is recovered by RecoverPanicAsError
func panicAfterNilPointerResult() (err error) {
	defer RecoverPanicAsError(&err)

	err = (*ptrError)(nil)
	panic("boom")
}

// panicAfterWrappedNilPointerResult panics after setting its result
// to a nil pointer error that WrapWithFuncParams wraps before
// RecoverPanicAsError recovers the panic
func panicAfterWrappedNilPointerResult(x int) (err error) {
	defer RecoverPanicAsError(&err)
	defer WrapWithFuncParams(&err, x)

	err = (*ptrError)(nil)
	panic("boom")
}

// fieldError is an error whose Error method panics
// if its field err is nil
type fieldError struct{ err error }

func (e *fieldError) Error() string { return "field: " + e.err.Error() }

// panicAfterPanickingResult panics after setting its result
// to an error whose Error method panics
func panicAfterPanickingResult() (err error) {
	defer RecoverPanicAsError(&err)

	err = &fieldError{}
	panic("boom")
}

// attemptsError wraps an error with a message prefix
// that changes when attempts is changed
type attemptsError struct {
	attempts int
	err      error
}

func (e *attemptsError) Error() string {
	return fmt.Sprintf("after %d attempts: %s", e.attempts, e.err)
}

func (e *attemptsError) Unwrap() error { return e.err }

// errorWithChangedWrapper returns an error whose message contains
// the message of a wrapper that changed after it was wrapped
func errorWithChangedWrapper() error {
	wrapper := &attemptsError{attempts: 1, err: loginWithFuncParams("alice", "hunter2")}
	err := fmt.Errorf("request failed: %w", wrapper)
	wrapper.attempts = 2
	return err
}

// errorWithChangedJoinSibling returns an error whose message contains
// the message of errors.Join with a sibling that changed after wrapping
func errorWithChangedJoinSibling() error {
	sibling := &attemptsError{attempts: 1, err: Sentinel("timeout")}
	err := fmt.Errorf("batch: %w", errors.Join(loginWithFuncParams("alice", "hunter2"), sibling))
	sibling.attempts = 2
	return err
}

// firstOfErrors is a multi-error that renders only its first error
type firstOfErrors []error

func (e firstOfErrors) Error() string   { return e[0].Error() }
func (e firstOfErrors) Unwrap() []error { return e }

// loginWithFuncParams returns an error wrapped with its call stack
// and its function parameters, the password kept secret
func loginWithFuncParams(user, password string) (err error) {
	defer WrapWithFuncParams(&err, user, KeepSecret(password))

	return New("login failed")
}

// sameLineErrors returns n errors created on the same line,
// so their messages including the call stacks are identical
func sameLineErrors(n int) []error {
	errs := make([]error, n)
	for i := range errs {
		errs[i] = New("duplicate")
	}
	return errs
}

// queryWithContext returns an error wrapped with its call stack
// and a context and a query as function parameters
func queryWithContext(ctx context.Context, query string) (err error) {
	defer WrapWithFuncParams(&err, ctx, query)

	return New("invalid")
}

// counter is a mutable function parameter
type counter struct{ N int }

// incrementWithFuncParams returns an error wrapped with its call stack
// and a pointer parameter that the caller may modify afterwards
func incrementWithFuncParams(c *counter) (err error) {
	defer WrapWithFuncParams(&err, c)

	return New("overflow")
}

// panicWithFuncParams recovers its panic as error
// wrapped with the debug stack and its function parameter
func panicWithFuncParams(secret string) (err error) {
	defer RecoverPanicAsErrorWithFuncParams(&err, secret)

	panic("boom")
}

// panicAfterSettingResult sets its result error before panicking
// with an error that has a call stack
func panicAfterSettingResult() (err error) {
	defer RecoverPanicAsError(&err)

	err = New("first")
	panic(New("boom"))
}

func TestMessageWithoutCallStacks(t *testing.T) {
	inner := New("inner")
	dups := sameLineErrors(2)
	require.Equal(t, dups[0].Error(), dups[1].Error(), "errors created on the same line")

	// The context parameter renders differently after cancel
	ctx, cancel := context.WithCancel(context.Background())
	canceledParamErr := fmt.Errorf("service: %w", queryWithContext(ctx, "SECRET QUERY"))
	cancel()

	// The pointer parameter renders differently after the modification
	c := &counter{N: 1}
	modifiedParamErr := Errorf("increment: %w", incrementWithFuncParams(c))
	c.N = 2

	tests := []struct {
		name string
		err  error
		want string
	}{
		{name: "nil", err: nil, want: ""},
		{name: "plain error", err: errors.New("plain"), want: "plain"},
		{name: "plain multi-line error", err: errors.New("line 1\nline 2"), want: "line 1\nline 2"},
		{name: "Sentinel", err: Sentinel("sentinel"), want: "sentinel"},
		{name: "New", err: New("failed"), want: "failed"},
		{name: "Errorf without %w", err: Errorf("value %d is invalid", 5), want: "value 5 is invalid"},
		{
			name: "Errorf wrapping an errs error",
			err:  Errorf("invalid from_date: %w", New("month out of range")),
			want: "invalid from_date: month out of range",
		},
		{
			name: "fmt.Errorf wrapping an errs error",
			err:  fmt.Errorf("invalid from_date: %w", New("month out of range")),
			want: "invalid from_date: month out of range",
		},
		{
			name: "two %w in one Errorf",
			err:  Errorf("first: %w, second: %w", New("a"), New("b")),
			want: "first: a, second: b",
		},
		{
			name: "two %w with reordered arguments",
			err:  fmt.Errorf("%[2]w before %[1]w", New("a"), New("b")),
			want: "b before a",
		},
		{
			name: "three levels of Errorf",
			err:  Errorf("handler: %w", fmt.Errorf("service: %w", Errorf("store: %w", New("timeout")))),
			want: "handler: service: store: timeout",
		},
		{
			name: "WrapWithCallStack of fmt.Errorf wrapping an errs error",
			err:  WrapWithCallStack(fmt.Errorf("context: %w", New("cause"))),
			want: "context: cause",
		},
		{
			name: "WrapWithFuncParams with KeepSecret",
			err:  loginWithFuncParams("alice", "hunter2"),
			want: "login failed",
		},
		{
			name: "Errorf wrapping WrapWithFuncParams",
			err:  Errorf("authenticate: %w", loginWithFuncParams("alice", "hunter2")),
			want: "authenticate: login failed",
		},
		{
			name: "errors.Join of errs errors including a nested one",
			err:  errors.Join(New("a"), Errorf("b: %w", New("c")), errors.New("d")),
			want: "a\nb: c\nd",
		},
		{
			name: "Errorf wrapping errors.Join",
			err:  Errorf("validation: %w", errors.Join(New("name missing"), New("email missing"))),
			want: "validation: name missing\nemail missing",
		},
		{name: "errors.Join of the same error twice", err: errors.Join(inner, inner), want: "inner\ninner"},
		{
			name: "errors.Join of identical errors",
			err:  errors.Join(dups[0], dups[1]),
			want: "duplicate\nduplicate",
		},
		{
			name: "errors.Join of a plain and an errs error with the same text",
			err:  errors.Join(Sentinel("same"), New("same")),
			want: "same\nsame",
		},
		{
			name: "errors.Join with a wrapper of the second error first",
			err:  errors.Join(Errorf("context: %w", inner), inner),
			want: "context: inner\ninner",
		},
		{
			name: "two %w reordered, the first contained in the second",
			err:  fmt.Errorf("%[2]w / %[1]w", inner, Errorf("context: %w", inner)),
			want: "context: inner / inner",
		},
		{
			name: "multi-error with a nil error rendering the same error once",
			err:  firstOfErrors{inner, nil, inner},
			want: "inner",
		},
		{
			// fmt.Errorf recovers the panic of a nil pointer's Error method
			// and renders "<nil>", so the wrapped nil pointer must be skipped
			name: "fmt.Errorf wrapping a nil pointer error",
			err:  fmt.Errorf("wrapped: %w", (*ptrError)(nil)),
			want: "wrapped: <nil>",
		},
		{
			name: "nil pointer error beside an errs error",
			err:  fmt.Errorf("%w / %w", (*ptrError)(nil), New("cause")),
			want: "<nil> / cause",
		},
		{
			name: "context parameter canceled after wrapping",
			err:  canceledParamErr,
			want: "service: invalid",
		},
		{
			name: "pointer parameter modified after wrapping",
			err:  modifiedParamErr,
			want: "increment: overflow",
		},
		{
			name: "same error wrapped with %w and formatted with %v",
			err:  fmt.Errorf("%w / %v", inner, inner),
			want: "inner / inner",
		},
		{name: "AsErrorWithDebugStack", err: AsErrorWithDebugStack("boom"), want: "boom"},
		{
			name: "AsErrorWithDebugStack of an errs error",
			err:  AsErrorWithDebugStack(New("boom")),
			want: "boom",
		},
		{
			name: "RecoverPanicAsErrorWithFuncParams",
			err:  panicWithFuncParams("SECRET PARAM"),
			want: "boom",
		},
		{
			name: "Errorf wrapping RecoverPanicAsErrorWithFuncParams",
			err:  Errorf("handler: %w", panicWithFuncParams("SECRET PARAM")),
			want: "handler: boom",
		},
		{
			name: "RecoverPanicAsError after setting the result",
			err:  panicAfterSettingResult(),
			want: "function returning error (first) panicked with: boom",
		},
		{name: "nil pointer error", err: (*ptrError)(nil), want: "<nil>"},
		{
			// the hook of the panic error type cleans the previous result,
			// which must not call Error on a nil pointer
			name: "RecoverPanicAsError after a nil pointer result",
			err:  panicAfterNilPointerResult(),
			want: "function returning error (<nil>) panicked with: boom",
		},
		{
			name: "AsErrorWithDebugStack of a nil pointer error",
			err:  fmt.Errorf("x: %w", AsErrorWithDebugStack(error((*ptrError)(nil)))),
			want: "x: <nil>",
		},
		{
			name: "call-stack wrapper of a nil pointer error",
			err:  WrapWithCallStack((*ptrError)(nil)),
			want: "<nil>",
		},
		{
			// WrapWithFuncParams wraps the nil pointer result before
			// RecoverPanicAsError keeps it as the previous result
			name: "RecoverPanicAsError after a wrapped nil pointer result",
			err:  panicAfterWrappedNilPointerResult(1),
			want: "function returning error (<nil>) panicked with: boom",
		},
		{
			// fmt.Errorf recovers the panic of the wrapper's Error method
			// and renders it into the message, which has no call stack
			name: "fmt.Errorf wrapping a call-stack wrapper of a nil pointer error",
			err:  fmt.Errorf("x: %w", WrapWithCallStack((*ptrError)(nil))),
			want: "x: %!v(PANIC=Error method: runtime error: invalid memory address or nil pointer dereference)",
		},
		{
			// fmt recovers the panic of an Error method
			// and renders it into the message
			name: "error whose Error method panics",
			err:  &fieldError{},
			want: "%!v(PANIC=Error method: runtime error: invalid memory address or nil pointer dereference)",
		},
		{
			name: "fmt.Errorf wrapping an error whose Error method panics",
			err:  fmt.Errorf("ctx: %w", &fieldError{}),
			want: "ctx: %!v(PANIC=Error method: runtime error: invalid memory address or nil pointer dereference)",
		},
		{
			name: "fmt.Errorf wrapping a call-stack wrapper of an error whose Error method panics",
			err:  fmt.Errorf("ctx: %w", WrapWithCallStack(&fieldError{})),
			want: "ctx: %!v(PANIC=Error method: runtime error: invalid memory address or nil pointer dereference)",
		},
		{
			name: "RecoverPanicAsError after a result whose Error method panics",
			err:  panicAfterPanickingResult(),
			want: "function returning error (%!v(PANIC=Error method: runtime error: invalid memory address or nil pointer dereference)) panicked with: boom",
		},
		{
			// the wrapper renders "after 2 attempts" now, but the call-stack
			// wrapper it wraps still renders the text the message contains
			name: "wrapper whose message changed after wrapping",
			err:  errorWithChangedWrapper(),
			want: "request failed: after 1 attempts: login failed",
		},
		{
			name: "errors.Join with a sibling whose message changed after wrapping",
			err:  errorWithChangedJoinSibling(),
			want: "batch: login failed\nafter 1 attempts: timeout",
		},
		{name: "ErrNotFound", err: WrapWithCallStack(ErrNotFound), want: "not found"},
		{
			name: "Errorf wrapping ErrNotFound",
			err:  Errorf("user %q: %w", "bob", ErrNotFound),
			want: `user "bob": not found`,
		},
		{
			name: "ReplaceErrNotFound with an errs error",
			err:  ReplaceErrNotFound(Errorf("load: %w", ErrNotFound), New("no such user")),
			want: "no such user",
		},
		{
			name: "ReplaceIfIs keeping an errs error",
			err:  ReplaceIfIs(Errorf("load: %w", New("timeout")), ErrNotFound, New("no such user")),
			want: "load: timeout",
		},
		{name: "DontLog", err: DontLog(Errorf("quiet: %w", New("cause"))), want: "quiet: cause"},
		{
			name: "pass-through wrapper",
			err:  errWrapper{Wrapped: Errorf("wrapped: %w", New("cause"))},
			want: "wrapped: cause",
		},
		{
			name: "reformatting wrapper of a plain error",
			err:  upperError{wrapped: errors.New("plain")},
			want: "PLAIN",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// given
			err := tt.err

			// when
			msg := MessageWithoutCallStacks(err)

			// then
			assert.Equal(t, tt.want, msg)
			assert.NotContains(t, msg, ".go:")
			assert.NotContains(t, msg, "alice")
			assert.NotContains(t, msg, "REDACTED")
			assert.NotContains(t, msg, "SECRET")
			assert.NotContains(t, msg, "goroutine")
		})
	}
}

// TestMessageWithoutCallStacks_ReformattingWrapper documents the limitation
// that a wrapper which does not contain the message of its wrapped error
// verbatim keeps the message it rendered, call stack included
func TestMessageWithoutCallStacks_ReformattingWrapper(t *testing.T) {
	// given
	err := upperError{wrapped: New("failed")}

	// when
	msg := MessageWithoutCallStacks(err)

	// then
	assert.Equal(t, err.Error(), msg)
	assert.Contains(t, msg, "FAILED\n")
	assert.Contains(t, msg, ".GO:")
}

// TestMessageWithoutCallStacks_FormattedWithV documents the limitation
// that an error formatted with %v instead of %w is not unwrappable,
// so its call stack stays in the message
func TestMessageWithoutCallStacks_FormattedWithV(t *testing.T) {
	// given
	err := Errorf("context: %v", New("cause"))

	// when
	msg := MessageWithoutCallStacks(err)

	// then
	assert.True(t, strings.HasPrefix(msg, "context: cause\n"), msg)
	assert.Contains(t, msg, "unwrap_test.go:")
}

// TestMessageWithoutCallStacks_PanicLogOutputUnchanged checks that
// the Error results of recovered panics still contain the debug stack
// and the function parameters for the logs
func TestMessageWithoutCallStacks_PanicLogOutputUnchanged(t *testing.T) {
	// given
	err := panicWithFuncParams("SECRET PARAM")

	// when
	logMsg := err.Error()

	// then
	assert.True(t, strings.HasPrefix(logMsg, "boom\ngoroutine "), logMsg)
	assert.Contains(t, logMsg, "SECRET PARAM")
	assert.Equal(t, "boom", MessageWithoutCallStacks(err))
}
