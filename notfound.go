package errs

import (
	"database/sql"
	"errors"
	"os"
)

// ErrNotFound is an universal error returned in case
// that a requested resource could not be found.
//
// Recommended usage:
//
// This error can be returned directly from a function
// if that function only requests one kind of resource
// and no further differentiation is needed about what
// resource could not be found.
//
// Else create custom "not found" error by wrapping ErrNotFound
// or implementing a custom error type with an
//   Is(target error) bool
// method that returns true for target == ErrNotFound.
//
// For checking errors it is recommended to use IsErrNotFound(err)
// instead of errors.Is(err, ErrNotFound) to also catch the
// standard library "not found" errors sql.ErrNoRows and os.ErrNotExist.
const ErrNotFound Sentinel = "not found"

// IsErrNotFound returns true if the passed error
// unwraps to, or is ErrNotFound, sql.ErrNoRows, or os.ErrNotExist.
func IsErrNotFound(err error) bool {
	return errors.Is(err, ErrNotFound) ||
		errors.Is(err, sql.ErrNoRows) ||
		errors.Is(err, os.ErrNotExist)
}

// ReplaceErrNotFound returns the passed replacement error
// if IsErrNotFound(err) returns true,
// meaning that all (optionally wrapped)
// ErrNotFound, sql.ErrNoRows, os.ErrNotExist
// errors get replaced.
func ReplaceErrNotFound(err, replacement error) error {
	if IsErrNotFound(err) {
		return replacement
	}
	return err
}
