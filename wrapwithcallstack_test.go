package errs

import (
	"reflect"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

// extractSentryPCs replicates the reflection-based program-counter
// extraction that sentry-go's ExtractStacktrace performs, so these tests
// can verify compatibility without adding a dependency on sentry-go.
//
// It probes the error's concrete type for the same method names Sentry
// looks for, in the same order (StackFrames, StackTrace, GetStackTracer),
// and reads program counters from the returned slice using the same rules:
// a bare uintptr element, or a struct with a ProgramCounter or PC field of
// kind uintptr.
func extractSentryPCs(err error) []uintptr {
	errValue := reflect.ValueOf(err)
	var method reflect.Value
	for _, name := range []string{"StackFrames", "StackTrace", "GetStackTracer"} {
		method = errValue.MethodByName(name)
		if method.IsValid() {
			break
		}
	}
	if !method.IsValid() {
		return nil
	}

	results := method.Call(nil)
	if len(results) != 1 || results[0].Kind() != reflect.Slice {
		return nil
	}
	stacktrace := results[0]

	var pcs []uintptr
	for i := 0; i < stacktrace.Len(); i++ {
		item := stacktrace.Index(i)
		switch item.Kind() {
		case reflect.Uintptr:
			pcs = append(pcs, uintptr(item.Uint()))
		case reflect.Struct:
			for _, field := range []string{"ProgramCounter", "PC"} {
				f := item.FieldByName(field)
				if f.IsValid() && f.Kind() == reflect.Uintptr {
					pcs = append(pcs, uintptr(f.Uint()))
					break
				}
			}
		}
	}
	return pcs
}

// frameFunctions resolves program counters to their function names the same
// way sentry-go does, via runtime.CallersFrames.
func frameFunctions(pcs []uintptr) []string {
	var names []string
	frames := runtime.CallersFrames(pcs)
	for {
		frame, more := frames.Next()
		names = append(names, frame.Function)
		if !more {
			break
		}
	}
	return names
}

// makeFuncParamsError returns an error whose concrete type is
// *withCallStackFuncParams, to exercise StackTrace promotion through the
// embedded withCallStack.
func makeFuncParamsError() (err error) {
	defer WrapWithFuncParams(&err, "param")

	return New("inner")
}

func TestStackTrace_SentryCompatible(t *testing.T) {
	err := New("boom") // concrete type *withCallStack, stack captured in this func
	require.IsType(t, &withCallStack{}, err)

	pcs := extractSentryPCs(err)
	require.NotEmpty(t, pcs, "Sentry-style reflection must extract program counters")

	names := frameFunctions(pcs)
	assert.Contains(t, names, "github.com/domonda/go-errs.TestStackTrace_SentryCompatible",
		"resolved stack must include the function that created the error")
}

func TestStackTrace_SentryCompatible_FuncParams(t *testing.T) {
	err := makeFuncParamsError()
	// Method is promoted from the embedded withCallStack.
	require.IsType(t, &withCallStackFuncParams{}, err)

	pcs := extractSentryPCs(err)
	require.NotEmpty(t, pcs, "Sentry-style reflection must extract program counters")

	names := frameFunctions(pcs)
	assert.Contains(t, names, "github.com/domonda/go-errs.makeFuncParamsError",
		"resolved stack must include the function that created the error")
}

func TestStackTrace_MatchesCallStack(t *testing.T) {
	withStack := New("boom").(*withCallStack)
	assert.Equal(t, withStack.CallStack(), withStack.StackTrace(),
		"StackTrace must return the same program counters as CallStack")

	funcParams := makeFuncParamsError().(*withCallStackFuncParams)
	assert.Equal(t, funcParams.CallStack(), funcParams.StackTrace(),
		"promoted StackTrace must return the same program counters as CallStack")
}
