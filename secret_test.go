package errs_test

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/domonda/go-errs"
)

func ExampleKeepSecret() {
	secret := errs.KeepSecret("My Password!")
	// Actual value:
	fmt.Println(secret.Secret())
	// Redacted string variants:
	fmt.Println(secret)
	fmt.Printf("%v\n", secret)
	fmt.Printf("%+v\n", secret)
	fmt.Printf("%#v\n", secret)

	// Output:
	// My Password!
	// ***REDACTED***
	// ***REDACTED***
	// ***REDACTED***
	// string(***REDACTED***)
}

func TestSecret_InFormatFunctionCall(t *testing.T) {
	secret := errs.KeepSecret("super-secret-password")

	// Test that FormatFunctionCall uses PrintForCallStack for secrets
	formatted := errs.FormatFunctionCall("TestFunc", "normalArg", secret, 42)

	require.Contains(t, formatted, "TestFunc")
	require.Contains(t, formatted, "normalArg")
	require.Contains(t, formatted, "***REDACTED***")
	require.Contains(t, formatted, "42")
	require.NotContains(t, formatted, "super-secret-password")
}

func TestSecret_InWrapWithFuncParams(t *testing.T) {
	secretValue := "super-secret-api-key"
	secret := errs.KeepSecret(secretValue)

	// Simulate a function that wraps an error with a secret parameter
	innerFunc := func(apiKey errs.Secret) (err error) {
		defer errs.WrapWithFuncParams(&err, apiKey)
		return errors.New("authentication failed")
	}

	err := innerFunc(secret)
	require.Error(t, err)

	errStr := err.Error()

	// The error message should contain REDACTED, not the actual secret
	require.Contains(t, errStr, "***REDACTED***")
	require.NotContains(t, errStr, secretValue)
	require.Contains(t, errStr, "authentication failed")
}

func TestSecret_InNestedWrapWithFuncParams(t *testing.T) {
	secretPassword := "my-secret-password-123"
	secret := errs.KeepSecret(secretPassword)

	// Simulate nested function calls with secret parameters
	innerFunc := func(password errs.Secret) (err error) {
		defer errs.WrapWithFuncParams(&err, password)
		return errors.New("invalid credentials")
	}

	outerFunc := func(userID string, password errs.Secret) (err error) {
		defer errs.WrapWithFuncParams(&err, userID, password)
		return innerFunc(password)
	}

	err := outerFunc("user123", secret)
	require.Error(t, err)

	errStr := err.Error()

	// The error message should never contain the actual secret
	require.NotContains(t, errStr, secretPassword)

	// Should contain REDACTED (possibly multiple times for nested calls)
	require.Contains(t, errStr, "***REDACTED***")

	// Should contain the non-secret parameters
	require.Contains(t, errStr, "user123")
	require.Contains(t, errStr, "invalid credentials")
}

func TestSecret_DifferentTypes(t *testing.T) {
	tests := []struct {
		name  string
		value any
	}{
		{"string", "secret-string"},
		{"int", 12345},
		{"slice", []byte("secret-bytes")},
		{"struct", struct{ Key string }{Key: "secret-key"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			secret := errs.KeepSecret(tt.value)

			// String should always be redacted
			require.Equal(t, "***REDACTED***", secret.String())

			// Original value should be retrievable via Secret()
			require.Equal(t, tt.value, secret.Secret())

			// FormatFunctionCall should use redacted value
			formatted := errs.FormatFunctionCall("Func", secret)
			require.Contains(t, formatted, "***REDACTED***")
			require.NotContains(t, formatted, fmt.Sprint(tt.value))
		})
	}
}
