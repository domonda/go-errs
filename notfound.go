package errs

import (
	"database/sql"
	"errors"
	"os"
)

// ErrNotFound is an universal error returned in case
// that a requested resource could not been found.
//
// Wrap this error to create custom "not found" errors
// and test them with IsErrNotFound instead of errors.Is(err, ErrNotFound)
// to also catch the standard library errors sql.ErrNoRows and os.ErrNotExist.
const ErrNotFound Sentinel = "not found"

// IsErrNotFound returns true if the passed error
// unwraps to, or is ErrNotFound, sql.ErrNoRows, or os.ErrNotExist.
func IsErrNotFound(err error) bool {
	return errors.Is(err, ErrNotFound) || errors.Is(err, sql.ErrNoRows) || errors.Is(err, os.ErrNotExist)
}
