package errs

import (
	"errors"
	"testing"
)

type stringer string

func (s stringer) String() string { return string(s) }

func TestAsError(t *testing.T) {
	tests := []struct {
		name    string
		input   interface{}
		wantErr error
	}{
		{name: "nil", input: nil, wantErr: nil},
		{name: "error", input: errors.New("error"), wantErr: errors.New("error")},
		{name: "errors", input: []error{errors.New("a"), errors.New("a")}, wantErr: Combine(errors.New("a"), errors.New("a"))},
		{name: "string", input: "string", wantErr: errors.New("string")},
		{name: "stringer", input: stringer("stringer"), wantErr: errors.New("stringer")},
		{name: "int", input: 666, wantErr: errors.New("666")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := AsError(tt.input)
			if ((err == nil) != (tt.wantErr == nil)) || (tt.wantErr != nil && err.Error() != tt.wantErr.Error()) {
				t.Errorf("AsError() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRecoverPanicAsError(t *testing.T) {
	f := func(p interface{}) (err error) {
		defer RecoverPanicAsError(&err)
		if p != nil {
			panic(p)
		}
		return nil
	}

	tests := []struct {
		name    string
		panic   interface{}
		wantErr error
	}{
		{name: "nil", panic: nil, wantErr: nil},
		// TODO normalize callstack because paths are on local machine
		// {name: "string", panic: "string", wantErr: errors.New("string")},
		// {name: "int", panic: 666, wantErr: errors.New("666")},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := f(tt.panic)
			if ((err == nil) != (tt.wantErr == nil)) || (tt.wantErr != nil && err.Error() != tt.wantErr.Error()) {
				t.Errorf("AsError() error = %s, wantErr %s", err, tt.wantErr)
			}
		})
	}
}
