package errs

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stringer string

func (s stringer) String() string { return string(s) }

func TestAsError(t *testing.T) {
	tests := []struct {
		name    string
		input   any
		wantErr error
	}{
		{name: "nil", input: nil, wantErr: nil},
		{name: "error", input: errors.New("error"), wantErr: errors.New("error")},
		{name: "errors", input: []error{errors.New("a"), errors.New("a")}, wantErr: errors.Join(errors.New("a"), errors.New("a"))},
		{name: "string", input: "string", wantErr: errors.New("string")},
		{name: "stringer", input: stringer("stringer"), wantErr: errors.New("stringer")},
		{name: "int", input: 666, wantErr: errors.New("666")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := AsError(tt.input)
			if ((err == nil) != (tt.wantErr == nil)) || (tt.wantErr != nil && err.Error() != tt.wantErr.Error()) {
				t.Errorf("AsError() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRecoverPanicAsError(t *testing.T) {
	f := func(p any) (err error) {
		defer RecoverPanicAsError(&err)
		if p != nil {
			panic(p)
		}
		return nil
	}

	t.Run("nil", func(t *testing.T) {
		assert.Nil(t, f(nil))
	})

	t.Run("string", func(t *testing.T) {
		err := f("string panic")
		require.NotNil(t, err)
		// Error() contains the debug.Stack() with machine-specific paths,
		// so compare the root error message after unwrapping the stack.
		assert.Equal(t, "string panic", Root(err).Error())
	})

	t.Run("int", func(t *testing.T) {
		err := f(666)
		require.NotNil(t, err)
		assert.Equal(t, "666", Root(err).Error())
	})

	t.Run("error", func(t *testing.T) {
		origErr := errors.New("original error")
		err := f(origErr)
		require.NotNil(t, err)
		assert.ErrorIs(t, err, origErr)
	})

	t.Run("panic after result was set", func(t *testing.T) {
		resultErr := errors.New("result")
		err := func() (err error) {
			defer RecoverPanicAsError(&err)
			err = resultErr
			panic("panic value")
		}()
		require.NotNil(t, err)
		assert.True(t, strings.HasPrefix(err.Error(), "function returning error (result) panicked with: panic value\n"), err.Error())
	})
}

func TestAsErrorWithDebugStack(t *testing.T) {
	assert.Nil(t, AsErrorWithDebugStack(nil))

	origErr := errors.New("original error")
	err := AsErrorWithDebugStack(origErr)
	assert.ErrorIs(t, err, origErr)
	// The debug stack follows the error message after a newline
	assert.True(t, strings.HasPrefix(err.Error(), "original error\ngoroutine "), err.Error())
}

func TestRecoverPanicAsErrorWithFuncParams(t *testing.T) {
	f := func(p any) (err error) {
		defer RecoverPanicAsErrorWithFuncParams(&err, "param", 666)
		if p != nil {
			panic(p)
		}
		return nil
	}

	t.Run("nil", func(t *testing.T) {
		assert.Nil(t, f(nil))
	})

	t.Run("string", func(t *testing.T) {
		err := f("panic value")
		require.NotNil(t, err)
		assert.True(t, strings.HasPrefix(err.Error(), "panic value\n"), err.Error())
		wrapped, ok := errors.AsType[*withCallStackFuncParams](err)
		require.True(t, ok)
		assert.Equal(t, []any{"param", 666}, wrapped.params)
	})

	t.Run("error", func(t *testing.T) {
		origErr := errors.New("original error")
		err := f(origErr)
		require.NotNil(t, err)
		assert.ErrorIs(t, err, origErr)
	})

	t.Run("panic after result was set", func(t *testing.T) {
		resultErr := errors.New("result")
		err := func() (err error) {
			defer RecoverPanicAsErrorWithFuncParams(&err, "param")
			err = resultErr
			panic("panic value")
		}()
		require.NotNil(t, err)
		assert.True(t, strings.HasPrefix(err.Error(), "function returning error (result) panicked with: panic value\n"), err.Error())
		wrapped, ok := errors.AsType[*withCallStackFuncParams](err)
		require.True(t, ok)
		assert.Equal(t, []any{"param"}, wrapped.params)
	})
}

func TestLogPanicWithFuncParams(t *testing.T) {
	f := func(log Logger, p any) {
		defer LogPanicWithFuncParams(log, "param", 666)
		if p != nil {
			panic(p)
		}
	}

	t.Run("no panic", func(t *testing.T) {
		log := &testLogger{}
		f(log, nil)
		assert.Empty(t, log.messages)
	})

	t.Run("panic is logged and re-panicked", func(t *testing.T) {
		log := &testLogger{}
		assert.PanicsWithValue(t, "panic value", func() { f(log, "panic value") })
		require.Len(t, log.messages, 1)
		assert.True(t, strings.HasPrefix(log.messages[0], "LogPanicWithFuncParams: panic value\n"), log.messages[0])
		assert.Contains(t, log.messages[0], "(`param`, 666)")
	})
}

func TestRecoverAndLogPanicWithFuncParams(t *testing.T) {
	f := func(log Logger, p any) {
		defer RecoverAndLogPanicWithFuncParams(log, "param", 666)
		if p != nil {
			panic(p)
		}
	}

	t.Run("no panic", func(t *testing.T) {
		log := &testLogger{}
		f(log, nil)
		assert.Empty(t, log.messages)
	})

	t.Run("panic is logged and recovered", func(t *testing.T) {
		log := &testLogger{}
		assert.NotPanics(t, func() { f(log, "panic value") })
		require.Len(t, log.messages, 1)
		assert.True(t, strings.HasPrefix(log.messages[0], "RecoverAndLogPanicWithFuncParams: panic value\n"), log.messages[0])
		assert.Contains(t, log.messages[0], "(`param`, 666)")
	})
}
