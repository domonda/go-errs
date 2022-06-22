package errs

import "testing"

func TestNew(t *testing.T) {
	err := New("test error")

	var nilError error
	if err == nilError {
		t.Fatal()
	}

	// Check against panic that happend when using error implementing
	// type withCallStack instead of *withCallStack:
	// comparing uncomparable type errs.withCallStack
	wrappedError := Errorf("wrapped: %w", errWrapper{Wrapped: errStruct{"a"}})
	if err == wrappedError {
		t.Fatal()
	}
}
