package errs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWrapWithCallStackSkip_Nil(t *testing.T) {
	err := WrapWithCallStackSkip(0, nil)
	assert.Nil(t, err, "WrapWithCallStackSkip with nil error should return nil")
}

func TestWrapWithCallStack_Nil(t *testing.T) {
	err := WrapWithCallStack(nil)
	assert.Nil(t, err, "WrapWithCallStack with nil error should return nil")
}

func TestNew(t *testing.T) {
	err := New("test error")

	var nilError error
	if err == nilError {
		t.Fatal()
	}

	// Check against panic that happened when using error implementing
	// type withCallStack instead of *withCallStack:
	// comparing uncomparable type errs.withCallStack
	wrappedError := Errorf("wrapped: %w", errWrapper{Wrapped: errStruct{"a"}})
	if err == wrappedError {
		t.Fatal()
	}
}
