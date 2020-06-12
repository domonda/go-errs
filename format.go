package errs

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"runtime"
	"strings"

	"github.com/domonda/go-pretty"
)

func formatError(err error) string {
	var (
		firstWithoutStack error
		calls             []string
	)

	for err != nil {
		switch e := err.(type) {
		case callStackParamsProvider:
			calls = append(calls, formatCallStackParams(e))

		case callStackProvider:
			calls = append(calls, formatCallStack(e))

		default:
			if firstWithoutStack == nil {
				firstWithoutStack = err
			}
		}

		err = errors.Unwrap(err)
	}

	if firstWithoutStack == nil {
		// Should never happen, just to make sure we don't panic
		firstWithoutStack = errors.New("wraperr: no wrapped error found")
	}

	var b strings.Builder
	fmt.Fprintln(&b, firstWithoutStack.Error())
	for i := len(calls) - 1; i >= 0; i-- {
		fmt.Fprintln(&b, calls[i])
	}
	return b.String()
}

func formatCallStack(e callStackProvider) string {
	stack := e.CallStack()
	frame, ok := runtime.CallersFrames(stack).Next()
	if !ok {
		return "insufficient call stack"
	}
	return fmt.Sprintf(
		"%s\n    %s:%d",
		frame.Function,
		frame.File,
		frame.Line,
	)
}

func formatCallStackParams(e callStackParamsProvider) string {
	stack, params := e.CallStackParams()
	frame, ok := runtime.CallersFrames(stack).Next()
	if !ok {
		return "insufficient call stack"
	}
	return fmt.Sprintf(
		"%s(%s)\n    %s:%d",
		frame.Function,
		formatParams(params),
		frame.File,
		frame.Line,
	)
}

func formatParams(params []interface{}) string {
	var b strings.Builder
	for i, param := range params {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(formatParam(param))
	}
	return b.String()
}

func formatParam(param interface{}) string {
	v := reflect.ValueOf(param)
	if isNil(v) {
		return "nil"
	}

	// Types typically not passed as pointers to type
	switch x := param.(type) {
	case error:
		return fmt.Sprintf("error(%q)", x.Error())

	case context.Context:
		return fmt.Sprintf("Context{Err:%v}", x.Err())

	case fmt.Stringer:
		return fmt.Sprintf("%q", x.String())

	case []byte:
		if len(x) > 300 {
			return fmt.Sprintf("[%d]byte(%q)", len(x), string(x[:20])+"...")
		}
		return fmt.Sprintf("[]byte(%q)", x)
	}

	// Dereference pointers
	for v.Kind() == reflect.Ptr && !v.IsNil() {
		v = v.Elem()
	}
	if isNil(v) {
		return "nil"
	}

	switch v.Kind() {
	case reflect.Func:
		return "func"

	case reflect.Struct, reflect.Map, reflect.Chan:
		return pretty.Sprint(v.Interface())

	case reflect.Array, reflect.Slice:
		var b strings.Builder
		fmt.Fprintf(&b, "%s{", v.Type())
		for i := 0; i < v.Len(); i++ {
			if i > 0 {
				b.WriteString(", ")
			}
			b.WriteString(formatParam(v.Index(i).Interface()))
		}
		b.WriteByte('}')
		return b.String()

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		// %#v would produce hex number for uint so use %d
		return fmt.Sprintf("%d", v.Interface())

	default:
		return fmt.Sprintf("%#v", v.Interface())
	}
}

// isNil returns if val is of a type that can be nil and if it is nil.
// Unlike reflect.Value.IsNil() it is safe to call this function for any value and type.
// The zero value of reflect.Value will yield true
// because it can be the result of reflect.ValueOf(nil)
func isNil(v reflect.Value) bool {
	if !v.IsValid() {
		return true
	}
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.Interface, reflect.Slice:
		return v.IsNil()
	}
	return false
}
