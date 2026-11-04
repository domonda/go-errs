package errs

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// errIsTarget matches target only through its Is method,
// without wrapping it
type errIsTarget struct{ target error }

func (e errIsTarget) Error() string        { return "is target" }
func (e errIsTarget) Is(target error) bool { return target == e.target }

func TestReplaceIfIs(t *testing.T) {
	target := errors.New("target")
	other := errors.New("other")
	replacement := errors.New("replacement")
	wrappedOther := fmt.Errorf("wrapped: %w", other)
	joinedOther := errors.Join(other, replacement)

	tests := []struct {
		name string
		err  error
		want error
	}{
		{name: "nil", err: nil, want: nil},
		{name: "target", err: target, want: replacement},
		{name: "wrapped target", err: fmt.Errorf("wrapped: %w", target), want: replacement},
		{name: "joined target", err: errors.Join(other, target), want: replacement},
		{name: "Is method matches target", err: errIsTarget{target}, want: replacement},
		{name: "other", err: other, want: other},
		{name: "wrapped other", err: wrappedOther, want: wrappedOther},
		{name: "joined other", err: joinedOther, want: joinedOther},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ReplaceIfIs(tt.err, target, replacement))
		})
	}

	// errors.Is(nil, nil) is true, but a nil err must still not be replaced
	assert.Nil(t, ReplaceIfIs(nil, nil, replacement))
}

func TestReplaceIfIsNot(t *testing.T) {
	target := errors.New("target")
	other := errors.New("other")
	replacement := errors.New("replacement")
	wrappedTarget := fmt.Errorf("wrapped: %w", target)
	joinedTarget := errors.Join(other, target)
	isTarget := errIsTarget{target}

	tests := []struct {
		name string
		err  error
		want error
	}{
		{name: "nil", err: nil, want: nil},
		{name: "target", err: target, want: target},
		{name: "wrapped target", err: wrappedTarget, want: wrappedTarget},
		{name: "joined target", err: joinedTarget, want: joinedTarget},
		{name: "Is method matches target", err: isTarget, want: isTarget},
		{name: "other", err: other, want: replacement},
		{name: "wrapped other", err: fmt.Errorf("wrapped: %w", other), want: replacement},
		{name: "joined other", err: errors.Join(other, replacement), want: replacement},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ReplaceIfIsNot(tt.err, target, replacement))
		})
	}
}

func TestReplaceIfType(t *testing.T) {
	typed := errStruct{Err: "typed"}
	other := errors.New("other")
	replacement := errors.New("replacement")
	wrappedOther := fmt.Errorf("wrapped: %w", other)
	noMatchAs := customAsError{msg: "as", matchType: "none"}

	tests := []struct {
		name string
		err  error
		want error
	}{
		{name: "nil", err: nil, want: nil},
		{name: "typed", err: typed, want: replacement},
		{name: "wrapped typed", err: fmt.Errorf("wrapped: %w", typed), want: replacement},
		{name: "joined typed", err: errors.Join(other, typed), want: replacement},
		{name: "As method matches type", err: customAsError{msg: "as", matchType: "errStruct"}, want: replacement},
		{name: "other", err: other, want: other},
		{name: "wrapped other", err: wrappedOther, want: wrappedOther},
		{name: "As method does not match type", err: noMatchAs, want: noMatchAs},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ReplaceIfType[errStruct](tt.err, replacement))
		})
	}
}

func TestReplaceIfNotType(t *testing.T) {
	typed := errStruct{Err: "typed"}
	other := errors.New("other")
	replacement := errors.New("replacement")
	wrappedTyped := fmt.Errorf("wrapped: %w", typed)
	joinedTyped := errors.Join(other, typed)
	matchAs := customAsError{msg: "as", matchType: "errStruct"}

	tests := []struct {
		name string
		err  error
		want error
	}{
		{name: "nil", err: nil, want: nil},
		{name: "typed", err: typed, want: typed},
		{name: "wrapped typed", err: wrappedTyped, want: wrappedTyped},
		{name: "joined typed", err: joinedTyped, want: joinedTyped},
		{name: "As method matches type", err: matchAs, want: matchAs},
		{name: "other", err: other, want: replacement},
		{name: "wrapped other", err: fmt.Errorf("wrapped: %w", other), want: replacement},
		{name: "As method does not match type", err: customAsError{msg: "as", matchType: "none"}, want: replacement},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ReplaceIfNotType[errStruct](tt.err, replacement))
		})
	}
}

func TestReplaceErrNotFound(t *testing.T) {
	other := errors.New("other")
	replacement := errors.New("replacement")

	tests := []struct {
		name string
		err  error
		want error
	}{
		{name: "nil", err: nil, want: nil},
		{name: "ErrNotFound", err: ErrNotFound, want: replacement},
		{name: "wrapped ErrNotFound", err: fmt.Errorf("user %w", ErrNotFound), want: replacement},
		{name: "sql.ErrNoRows", err: sql.ErrNoRows, want: replacement},
		{name: "wrapped sql.ErrNoRows", err: fmt.Errorf("query: %w", sql.ErrNoRows), want: replacement},
		{name: "os.ErrNotExist", err: os.ErrNotExist, want: replacement},
		{name: "wrapped os.ErrNotExist", err: fmt.Errorf("open: %w", os.ErrNotExist), want: replacement},
		{name: "other", err: other, want: other},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, ReplaceErrNotFound(tt.err, replacement))
		})
	}
}
