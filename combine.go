package errs

import "errors"

// Combine returns a MultiError error for 2 or more errors that are not nil,
// or the same error if only one error was passed,
// or nil if zero arguments are passed or all passed errors are nil.
//
// The MultiError Error method returns the strings from the
// individual Error methods joined by the new line character '\n'.
//
// In case of a MultiError, errors.Is and errors.As will return true
// for the first matched error.
//
// Combine does not wrap the passed errors with a text or call stack.
//
// The motivation behind Combine and MultiError is to combine different
// logical errors into one, as compared to error wrapping
// which adds more information to one logical error.
func Combine(errs ...error) error {
	var combined multiError
	for _, err := range errs {
		if err != nil {
			var m multiError
			if errors.As(err, &m) {
				combined = append(combined, m.Errors()...)
			} else {
				combined = append(combined, err)
			}
		}
	}

	switch len(combined) {
	case 0:
		return nil
	case 1:
		return combined[0]
	default:
		return combined
	}
}

// Uncombine returns multiple errors if err is a MultiError,
// else it will return a single element slice containing err
// or nil if err is nil.
func Uncombine(err error) []error {
	if err == nil {
		return nil
	}
	var multi MultiError
	if errors.As(err, &multi) {
		return multi.Errors()
	}
	return []error{err}
}
