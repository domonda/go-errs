package errs

import (
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/domonda/go-pretty"
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

// Test PrintFuncFor customization

type ThirdPartyAPIKey string

type CustomStringer struct {
	Name string
}

func (c CustomStringer) String() string {
	return "CustomStringer[" + c.Name + "]"
}

type SensitiveData struct {
	PublicID  string
	SecretKey string
}

func TestPrintFuncFor_MaskSensitiveStrings(t *testing.T) {
	// Save original printer
	originalPrinter := Printer
	defer func() { Printer = originalPrinter }()

	// Configure printer to mask strings containing sensitive keywords
	Printer = Printer.WithPrintFuncFor(func(v reflect.Value) pretty.PrintFunc {
		if v.Kind() == reflect.String {
			str := v.String()
			if strings.Contains(strings.ToLower(str), "password") ||
				strings.Contains(strings.ToLower(str), "token") ||
				strings.Contains(strings.ToLower(str), "apikey") {
				return func(w io.Writer) {
					io.WriteString(w, "`***REDACTED***`")
				}
			}
		}
		return pretty.PrintFuncForPrintable(v)
	})

	result := FormatFunctionCall("testFunc", "my-password-123", "safe-string", "Bearer-token-xyz")

	assert.Contains(t, result, "testFunc(")
	assert.Contains(t, result, "`***REDACTED***`") // password should be masked
	assert.Contains(t, result, "`safe-string`")    // safe string should appear
	// Second redacted for token
	occurrences := strings.Count(result, "`***REDACTED***`")
	assert.Equal(t, 2, occurrences, "Should have 2 redacted strings")
	assert.NotContains(t, result, "my-password-123")
	assert.NotContains(t, result, "Bearer-token-xyz")
}

func TestPrintFuncFor_CustomTypeRedaction(t *testing.T) {
	// Save original printer
	originalPrinter := Printer
	defer func() { Printer = originalPrinter }()

	// Configure printer to mask ThirdPartyAPIKey type
	Printer = Printer.WithPrintFuncFor(func(v reflect.Value) pretty.PrintFunc {
		if v.Type().String() == "errs.ThirdPartyAPIKey" {
			return func(w io.Writer) {
				io.WriteString(w, "***REDACTED_API_KEY***")
			}
		}
		return pretty.PrintFuncForPrintable(v)
	})

	apiKey := ThirdPartyAPIKey("sk-1234567890abcdef")
	result := FormatFunctionCall("testFunc", "user-123", apiKey)

	assert.Contains(t, result, "testFunc(")
	assert.Contains(t, result, "`user-123`")
	assert.Contains(t, result, "***REDACTED_API_KEY***")
	assert.NotContains(t, result, "sk-1234567890abcdef")
}

func TestPrintFuncFor_AdaptFmtStringer(t *testing.T) {
	// Save original printer
	originalPrinter := Printer
	defer func() { Printer = originalPrinter }()

	// Configure printer to use String() method from fmt.Stringer types
	Printer = Printer.WithPrintFuncFor(func(v reflect.Value) pretty.PrintFunc {
		stringer, ok := v.Interface().(fmt.Stringer)
		if !ok && v.CanAddr() {
			stringer, ok = v.Addr().Interface().(fmt.Stringer)
		}
		if ok {
			return func(w io.Writer) {
				io.WriteString(w, stringer.String())
			}
		}
		return pretty.PrintFuncForPrintable(v)
	})

	custom := CustomStringer{Name: "test"}
	result := FormatFunctionCall("testFunc", custom, "other")

	assert.Contains(t, result, "testFunc(")
	assert.Contains(t, result, "CustomStringer[test]")
	assert.Contains(t, result, "`other`")
}

func TestPrintFuncFor_StructFieldMasking(t *testing.T) {
	// Save original printer
	originalPrinter := Printer
	defer func() { Printer = originalPrinter }()

	// Configure printer to mask struct fields containing "secret" in name
	Printer = Printer.WithPrintFuncFor(func(v reflect.Value) pretty.PrintFunc {
		if v.Kind() == reflect.Struct {
			t := v.Type()
			hasSecretField := false
			for i := 0; i < t.NumField(); i++ {
				field := t.Field(i)
				if strings.Contains(strings.ToLower(field.Name), "secret") {
					hasSecretField = true
					break
				}
			}
			if hasSecretField {
				return func(w io.Writer) {
					io.WriteString(w, t.Name())
					io.WriteString(w, "{***FIELDS_REDACTED***}")
				}
			}
		}
		return pretty.PrintFuncForPrintable(v)
	})

	sensitive := SensitiveData{
		PublicID:  "public-123",
		SecretKey: "should-not-appear",
	}
	result := FormatFunctionCall("testFunc", sensitive)

	assert.Contains(t, result, "testFunc(")
	assert.Contains(t, result, "SensitiveData{***FIELDS_REDACTED***}")
	assert.NotContains(t, result, "public-123")
	assert.NotContains(t, result, "should-not-appear")
}

func TestPrintFuncFor_PreservesPrintableInterface(t *testing.T) {
	// Save original printer
	originalPrinter := Printer
	defer func() { Printer = originalPrinter }()

	// Configure printer with custom logic that falls back to Printable
	Printer = Printer.WithPrintFuncFor(func(v reflect.Value) pretty.PrintFunc {
		// Custom logic that doesn't match our types
		if v.Kind() == reflect.String && v.String() == "special" {
			return func(w io.Writer) {
				io.WriteString(w, "`SPECIAL`")
			}
		}
		// Fall back to default Printable interface handling
		return pretty.PrintFuncForPrintable(v)
	})

	simple := &SimplePrintable{Value: "test"}
	result := FormatFunctionCall("testFunc", simple, "special", "normal")

	assert.Contains(t, result, "testFunc(")
	// SimplePrintable should still use its PrettyPrint method
	assert.Contains(t, result, "Simple{test}")
	// "special" string should use custom logic
	assert.Contains(t, result, "`SPECIAL`")
	// "normal" string should use default formatting
	assert.Contains(t, result, "`normal`")
}

func TestPrintFuncFor_WithWrapWithFuncParams(t *testing.T) {
	// Save original printer
	originalPrinter := Printer
	defer func() { Printer = originalPrinter }()

	// Configure printer to mask sensitive data
	Printer = Printer.WithPrintFuncFor(func(v reflect.Value) pretty.PrintFunc {
		if v.Kind() == reflect.String {
			str := v.String()
			if strings.Contains(str, "secret-") {
				return func(w io.Writer) {
					io.WriteString(w, "`***REDACTED***`")
				}
			}
		}
		return pretty.PrintFuncForPrintable(v)
	})

	testFunc := func(userID string, apiKey string) (err error) {
		defer WrapWithFuncParams(&err, userID, apiKey)
		return New("operation failed")
	}

	err := testFunc("user-123", "secret-api-key-xyz")
	require.Error(t, err)

	errStr := err.Error()
	assert.Contains(t, errStr, "operation failed")
	assert.Contains(t, errStr, "`user-123`")
	assert.Contains(t, errStr, "`***REDACTED***`")
	assert.NotContains(t, errStr, "secret-api-key-xyz")
}

func TestPrintFuncFor_NestedStructsWithCustomFormatting(t *testing.T) {
	// Save original printer
	originalPrinter := Printer
	defer func() { Printer = originalPrinter }()

	type NestedSensitive struct {
		Data SensitiveData
		Name string
	}

	// Configure printer to mask SensitiveData type
	Printer = Printer.WithPrintFuncFor(func(v reflect.Value) pretty.PrintFunc {
		if v.Type().String() == "errs.SensitiveData" {
			return func(w io.Writer) {
				io.WriteString(w, "SensitiveData{***MASKED***}")
			}
		}
		return pretty.PrintFuncForPrintable(v)
	})

	nested := NestedSensitive{
		Data: SensitiveData{
			PublicID:  "public",
			SecretKey: "secret",
		},
		Name: "container",
	}

	result := FormatFunctionCall("testFunc", nested)

	assert.Contains(t, result, "testFunc(")
	assert.Contains(t, result, "NestedSensitive{")
	assert.Contains(t, result, "Data:SensitiveData{***MASKED***}")
	assert.Contains(t, result, "Name:`container`")
	assert.NotContains(t, result, "public")
	assert.NotContains(t, result, "secret")
}

func TestPrintFuncFor_NilReturnUsesDefault(t *testing.T) {
	// Save original printer
	originalPrinter := Printer
	defer func() { Printer = originalPrinter }()

	// Configure printer that returns nil for non-matching cases
	Printer = Printer.WithPrintFuncFor(func(v reflect.Value) pretty.PrintFunc {
		if v.Kind() == reflect.String && v.String() == "intercept" {
			return func(w io.Writer) {
				io.WriteString(w, "`INTERCEPTED`")
			}
		}
		// Return nil to use default behavior
		return nil
	})

	result := FormatFunctionCall("testFunc", "intercept", "normal", 42)

	assert.Contains(t, result, "testFunc(")
	assert.Contains(t, result, "`INTERCEPTED`")
	assert.Contains(t, result, "`normal`")
	assert.Contains(t, result, "42")
}
