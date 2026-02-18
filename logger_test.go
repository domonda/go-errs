package errs

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

type testDecisionMaker bool

func (e testDecisionMaker) Error() string {
	return fmt.Sprint(bool(e))
}

func (e testDecisionMaker) ShouldLog() bool {
	return bool(e)
}

func TestShouldLog(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"non LogDecisionMaker", errors.New("non LogDecisionMaker"), true},
		{"true", testDecisionMaker(true), true},
		{"false", testDecisionMaker(false), false},
		{"dont log", DontLog(testDecisionMaker(true)), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ShouldLog(tt.err); got != tt.want {
				t.Errorf("ShouldLog() = %v, want %v", got, tt.want)
			}
		})
	}
}

// testLogger captures Printf output for testing.
type testLogger struct {
	messages []string
}

func (l *testLogger) Printf(format string, args ...any) {
	l.messages = append(l.messages, fmt.Sprintf(format, args...))
}

func TestLogFunctionCall(t *testing.T) {
	t.Run("nil logger does not panic", func(t *testing.T) {
		assert.NotPanics(t, func() {
			LogFunctionCall(nil, "myFunc", "arg1", 42)
		})
	})

	t.Run("logs formatted call", func(t *testing.T) {
		log := &testLogger{}
		LogFunctionCall(log, "myFunc", "arg1", 42)

		assert.Len(t, log.messages, 1)
		assert.True(t, strings.Contains(log.messages[0], "myFunc("))
		assert.True(t, strings.Contains(log.messages[0], "arg1"))
		assert.True(t, strings.Contains(log.messages[0], "42"))
	})

	t.Run("no params", func(t *testing.T) {
		log := &testLogger{}
		LogFunctionCall(log, "noArgs")

		assert.Len(t, log.messages, 1)
		assert.Equal(t, "noArgs()", log.messages[0])
	})
}
