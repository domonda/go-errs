package errs

// import (
// 	"errors"
// 	"strings"
// )

// // MultiError combines multiple errors into one.
// // The Error method returns the strings from the individual Error methods
// // joined by the new line character '\n'.
// // The motivation behind Combine and MultiError is to combine different
// // logical errors into one, as compared to error wrapping,
// // which adds more information to one logical error.
// type MultiError interface {
// 	// Error implements the error interface.
// 	Error() string

// 	// Err returns the MultiError or nil
// 	// if it does not contain any errors.
// 	//
// 	// Note that an empty MultiError
// 	// still implements the error interface
// 	// with the Error method returning a string.
// 	// Always use the Err method to convert
// 	// MultiError to an error.
// 	Err() error

// 	// Errors returns the wrapped errors.
// 	Errors() []error
// }

// type multiError []error

// func (m multiError) Error() string {
// 	if len(m) == 0 {
// 		return "no error"
// 	}
// 	var b strings.Builder
// 	for i, err := range m {
// 		if i > 0 {
// 			b.WriteByte('\n')
// 		}
// 		b.WriteString(err.Error())
// 	}
// 	return b.String()
// }

// func (m multiError) Err() error {
// 	if len(m) == 0 {
// 		return nil
// 	}
// 	return m
// }

// func (m multiError) Errors() []error {
// 	return m
// }

// func (m multiError) Is(target error) bool {
// 	for _, err := range m {
// 		if errors.Is(err, target) {
// 			return true
// 		}
// 	}
// 	return false
// }

// func (m multiError) As(target any) bool {
// 	for _, err := range m {
// 		if errors.As(err, target) {
// 			return true
// 		}
// 	}
// 	return false
// }
