package errs

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test types that implement pretty.Printable

type SimplePrintable struct {
	Value string
}

func (s *SimplePrintable) PrettyPrint(w io.Writer) {
	io.WriteString(w, "Simple{")
	io.WriteString(w, s.Value)
	io.WriteString(w, "}")
}

type NestedPrintable struct {
	Value string
}

func (n *NestedPrintable) PrettyPrint(w io.Writer) {
	io.WriteString(w, "Nested{")
	io.WriteString(w, n.Value)
	io.WriteString(w, "}")
}

// Struct with nested pretty.Printable field
type ContainerWithNested struct {
	ID     string
	Nested *NestedPrintable
	Count  int
}

// Struct with multiple nested pretty.Printable fields
type MultiNestedContainer struct {
	First  *SimplePrintable
	Second *NestedPrintable
	Name   string
}

// Deeply nested structs
type DeepContainer struct {
	Inner *ContainerWithNested
	Label string
}

// Struct without pretty.Printable
type RegularStruct struct {
	Name  string
	Value int
}

func TestFormatFunctionCall_TopLevelPrintable(t *testing.T) {
	simple := &SimplePrintable{Value: "test"}

	result := FormatFunctionCall("testFunc", simple)

	assert.Contains(t, result, "testFunc(")
	assert.Contains(t, result, "Simple{test}")
}

func TestFormatFunctionCall_NestedPrintable(t *testing.T) {
	nested := &NestedPrintable{Value: "inner"}
	container := &ContainerWithNested{
		ID:     "container-1",
		Nested: nested,
		Count:  42,
	}

	result := FormatFunctionCall("testFunc", container)

	assert.Contains(t, result, "testFunc(")
	assert.Contains(t, result, "ContainerWithNested{")
	assert.Contains(t, result, "ID:`container-1`")
	assert.Contains(t, result, "Nested:Nested{inner}")
	assert.Contains(t, result, "Count:42")
}

func TestFormatFunctionCall_MultipleNestedPrintable(t *testing.T) {
	container := &MultiNestedContainer{
		First:  &SimplePrintable{Value: "first"},
		Second: &NestedPrintable{Value: "second"},
		Name:   "multi",
	}

	result := FormatFunctionCall("testFunc", container)

	assert.Contains(t, result, "testFunc(")
	assert.Contains(t, result, "MultiNestedContainer{")
	assert.Contains(t, result, "First:Simple{first}")
	assert.Contains(t, result, "Second:Nested{second}")
	assert.Contains(t, result, "Name:`multi`")
}

func TestFormatFunctionCall_DeeplyNestedPrintable(t *testing.T) {
	deep := &DeepContainer{
		Inner: &ContainerWithNested{
			ID:     "deep-1",
			Nested: &NestedPrintable{Value: "deeply-nested"},
			Count:  99,
		},
		Label: "outer",
	}

	result := FormatFunctionCall("testFunc", deep)

	assert.Contains(t, result, "testFunc(")
	assert.Contains(t, result, "DeepContainer{")
	assert.Contains(t, result, "ContainerWithNested{")
	// The Nested field should use its PrettyPrint method
	assert.Contains(t, result, "Nested:Nested{deeply-nested}")
	assert.Contains(t, result, "Label:`outer`")
	assert.Contains(t, result, "ID:`deep-1`")
}

func TestFormatFunctionCall_MixedParameters(t *testing.T) {
	nested := &NestedPrintable{Value: "nested"}
	container := &ContainerWithNested{
		ID:     "mix-1",
		Nested: nested,
		Count:  10,
	}
	regular := &RegularStruct{Name: "regular", Value: 123}

	result := FormatFunctionCall("testFunc", container, "plain-string", 42, regular)

	assert.Contains(t, result, "testFunc(")
	assert.Contains(t, result, "ContainerWithNested{")
	assert.Contains(t, result, "Nested{nested}")
	assert.Contains(t, result, "`plain-string`")
	assert.Contains(t, result, "42")
	assert.Contains(t, result, "RegularStruct{")
}

func TestFormatFunctionCall_NilNestedField(t *testing.T) {
	container := &ContainerWithNested{
		ID:     "nil-test",
		Nested: nil,
		Count:  0,
	}

	result := FormatFunctionCall("testFunc", container)

	// Should not panic and should handle nil gracefully
	assert.Contains(t, result, "testFunc(")
	assert.Contains(t, result, "ContainerWithNested{")
	assert.Contains(t, result, "ID:`nil-test`")
}

func TestFormatFunctionCall_RegularStructWithoutPrintable(t *testing.T) {
	regular := &RegularStruct{Name: "test", Value: 456}

	result := FormatFunctionCall("testFunc", regular)

	// Should use default go-pretty formatting
	assert.Contains(t, result, "testFunc(")
	assert.Contains(t, result, "RegularStruct{")
	assert.Contains(t, result, "Name:`test`")
	assert.Contains(t, result, "Value:456")
}

func TestWrapWithFuncParams_NestedPrintable(t *testing.T) {
	testFunc := func(container *ContainerWithNested) (err error) {
		defer WrapWithFuncParams(&err, container)
		return New("test error")
	}

	nested := &NestedPrintable{Value: "wrapped"}
	container := &ContainerWithNested{
		ID:     "wrap-1",
		Nested: nested,
		Count:  7,
	}

	err := testFunc(container)
	require.Error(t, err)

	errStr := err.Error()
	assert.Contains(t, errStr, "test error")
	assert.Contains(t, errStr, "ContainerWithNested{")
	assert.Contains(t, errStr, "Nested{wrapped}")
}

// Test Secret handling

type ContainerWithSecret struct {
	Username string
	Password Secret
	APIKey   Secret
}

type NestedSecretContainer struct {
	ServiceName string
	Credentials *ContainerWithSecret
	Timeout     int
}

func TestFormatFunctionCall_SecretInStruct(t *testing.T) {
	container := &ContainerWithSecret{
		Username: "admin",
		Password: KeepSecret("super-secret-password"),
		APIKey:   KeepSecret("sk-1234567890abcdef"),
	}

	result := FormatFunctionCall("testFunc", container)

	assert.Contains(t, result, "testFunc(")
	assert.Contains(t, result, "ContainerWithSecret{")
	assert.Contains(t, result, "Username:`admin`")
	assert.Contains(t, result, "Password:***REDACTED***")
	assert.Contains(t, result, "APIKey:***REDACTED***")
	// Ensure actual secrets are not in output
	assert.NotContains(t, result, "super-secret-password")
	assert.NotContains(t, result, "sk-1234567890abcdef")
}

func TestFormatFunctionCall_NestedSecret(t *testing.T) {
	nested := &NestedSecretContainer{
		ServiceName: "api-service",
		Credentials: &ContainerWithSecret{
			Username: "service-account",
			Password: KeepSecret("nested-secret-pass"),
			APIKey:   KeepSecret("nested-api-key-xyz"),
		},
		Timeout: 30,
	}

	result := FormatFunctionCall("testFunc", nested)

	assert.Contains(t, result, "testFunc(")
	assert.Contains(t, result, "NestedSecretContainer{")
	assert.Contains(t, result, "ServiceName:`api-service`")
	assert.Contains(t, result, "ContainerWithSecret{")
	assert.Contains(t, result, "Username:`service-account`")
	assert.Contains(t, result, "Password:***REDACTED***")
	assert.Contains(t, result, "APIKey:***REDACTED***")
	assert.Contains(t, result, "Timeout:30")
	// Ensure actual secrets are not in output
	assert.NotContains(t, result, "nested-secret-pass")
	assert.NotContains(t, result, "nested-api-key-xyz")
}

func TestFormatFunctionCall_MixedPrintableAndSecret(t *testing.T) {
	type MixedContainer struct {
		Doc      *SimplePrintable
		Secret   Secret
		Metadata string
	}

	container := &MixedContainer{
		Doc:      &SimplePrintable{Value: "document-123"},
		Secret:   KeepSecret("secret-token"),
		Metadata: "public-info",
	}

	result := FormatFunctionCall("testFunc", container)

	assert.Contains(t, result, "testFunc(")
	assert.Contains(t, result, "MixedContainer{")
	assert.Contains(t, result, "Doc:Simple{document-123}")
	assert.Contains(t, result, "Secret:***REDACTED***")
	assert.Contains(t, result, "Metadata:`public-info`")
	// Ensure secret is not in output
	assert.NotContains(t, result, "secret-token")
}

func TestWrapWithFuncParams_SecretInNestedStruct(t *testing.T) {
	testFunc := func(container *NestedSecretContainer) (err error) {
		defer WrapWithFuncParams(&err, container)
		return New("authentication failed")
	}

	nested := &NestedSecretContainer{
		ServiceName: "auth-service",
		Credentials: &ContainerWithSecret{
			Username: "user",
			Password: KeepSecret("should-not-appear"),
			APIKey:   KeepSecret("sk-should-not-appear"),
		},
		Timeout: 60,
	}

	err := testFunc(nested)
	require.Error(t, err)

	errStr := err.Error()
	assert.Contains(t, errStr, "authentication failed")
	assert.Contains(t, errStr, "NestedSecretContainer{")
	assert.Contains(t, errStr, "Username:`user`")
	assert.Contains(t, errStr, "Password:***REDACTED***")
	assert.Contains(t, errStr, "APIKey:***REDACTED***")
	// Most importantly: ensure secrets are NOT in the error message
	assert.NotContains(t, errStr, "should-not-appear")
	assert.NotContains(t, errStr, "sk-should-not-appear")
}

func TestFormatFunctionCall_TopLevelSecret(t *testing.T) {
	// Test that a top-level secret parameter is handled
	secret := KeepSecret("top-level-secret")

	result := FormatFunctionCall("testFunc", "username", secret)

	assert.Contains(t, result, "testFunc(")
	assert.Contains(t, result, "`username`")
	assert.Contains(t, result, "***REDACTED***")
	assert.NotContains(t, result, "top-level-secret")
}
