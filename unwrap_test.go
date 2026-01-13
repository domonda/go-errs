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
	msg := "multi:"
	for i, err := range e.errs {
		if i > 0 {
			msg += ","
		}
		msg += err.Error()
	}
	return msg
}

func (e multiWrapper) Unwrap() []error { return e.errs }

// multiWrapperWithAs implements both Unwrap() []error and As(any) bool
type multiWrapperWithAs struct {
	errs      []error
	matchType string
}

func (e multiWrapperWithAs) Error() string { return "multiWithAs" }
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
	for i := 0; i < depth; i++ {
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
