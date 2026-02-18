package errs

import (
	"errors"
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
}
