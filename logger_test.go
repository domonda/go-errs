package errs

import (
	"errors"
	"fmt"
	"testing"
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
		{"non LogDecisionMaker", errors.New("non LogDecisionMaker"), false},
		{"true", testDecisionMaker(true), true},
		{"false", testDecisionMaker(false), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ShouldLog(tt.err); got != tt.want {
				t.Errorf("ShouldLog() = %v, want %v", got, tt.want)
			}
		})
	}
}
