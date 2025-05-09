package errs

import "iter"

// IterSeq returns an iter.Seq[error] iterator
// that yields the passed error once.
//
// See the [iter] package documentation for more details.
func IterSeq(err error) iter.Seq[error] {
	return func(yield func(error) bool) {
		yield(err)
	}
}

// IterSeq2 returns an iter.Seq2[T, error] iterator
// that yields the default value of T and the passed error once.
//
// This is useful for the design-pattern of using first value of a
// two value iterator as actual value and the second as error.
//
// See the [iter] package documentation for more details.
func IterSeq2[T any](err error) iter.Seq2[T, error] {
	return func(yield func(T, error) bool) {
		yield(*new(T), err)
	}
}
