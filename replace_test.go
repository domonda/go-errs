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

// TestReplaceIfIsAndIsNot checks both functions against the same cases,
// ReplaceIfIsNot must replace exactly where ReplaceIfIs does not
func TestReplaceIfIsAndIsNot(t *testing.T) {
	target := errors.New("target")
	other := errors.New("other")
	replacement := errors.New("replacement")

	tests := []struct {
		name  string
		err   error
		match bool
	}{
		{name: "target", err: target, match: true},
		{name: "wrapped target", err: fmt.Errorf("wrapped: %w", target), match: true},
		{name: "joined target", err: errors.Join(other, target), match: true},
		{name: "Is method matches target", err: errIsTarget{target}, match: true},
		{name: "other", err: other, match: false},
		{name: "wrapped other", err: fmt.Errorf("wrapped: %w", other), match: false},
		{name: "joined other", err: errors.Join(other, replacement), match: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.match {
				assert.Equal(t, replacement, ReplaceIfIs(tt.err, target, replacement))
				assert.Equal(t, tt.err, ReplaceIfIsNot(tt.err, target, replacement))
			} else {
				assert.Equal(t, tt.err, ReplaceIfIs(tt.err, target, replacement))
				assert.Equal(t, replacement, ReplaceIfIsNot(tt.err, target, replacement))
			}
		})
	}

	assert.Nil(t, ReplaceIfIs(nil, target, replacement))
	assert.Nil(t, ReplaceIfIsNot(nil, target, replacement))
	// errors.Is(nil, nil) is true, but a nil err must still not be replaced
	assert.Nil(t, ReplaceIfIs(nil, nil, replacement))
}

// TestReplaceIfTypeAndNotType checks both functions against the same cases,
// ReplaceIfNotType must replace exactly where ReplaceIfType does not
func TestReplaceIfTypeAndNotType(t *testing.T) {
	typed := errStruct{Err: "typed"}
	other := errors.New("other")
	replacement := errors.New("replacement")

	tests := []struct {
		name  string
		err   error
		match bool
	}{
		{name: "typed", err: typed, match: true},
		{name: "wrapped typed", err: fmt.Errorf("wrapped: %w", typed), match: true},
		{name: "joined typed", err: errors.Join(other, typed), match: true},
		{name: "As method matches type", err: customAsError{msg: "as", matchType: "errStruct"}, match: true},
		{name: "other", err: other, match: false},
		{name: "wrapped other", err: fmt.Errorf("wrapped: %w", other), match: false},
		{name: "As method does not match type", err: customAsError{msg: "as", matchType: "none"}, match: false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.match {
				assert.Equal(t, replacement, ReplaceIfType[errStruct](tt.err, replacement))
				assert.Equal(t, tt.err, ReplaceIfNotType[errStruct](tt.err, replacement))
			} else {
				assert.Equal(t, tt.err, ReplaceIfType[errStruct](tt.err, replacement))
				assert.Equal(t, replacement, ReplaceIfNotType[errStruct](tt.err, replacement))
			}
		})
	}

	assert.Nil(t, ReplaceIfType[errStruct](nil, replacement))
	assert.Nil(t, ReplaceIfNotType[errStruct](nil, replacement))
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
