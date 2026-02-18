package errs

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIterSeq(t *testing.T) {
	t.Run("yields error once", func(t *testing.T) {
		err := errors.New("test error")
		var collected []error
		for e := range IterSeq(err) {
			collected = append(collected, e)
		}
		assert.Equal(t, []error{err}, collected)
	})

	t.Run("yields nil error once", func(t *testing.T) {
		var count int
		for e := range IterSeq(nil) {
			assert.Nil(t, e)
			count++
		}
		assert.Equal(t, 1, count)
	})
}

func TestIterSeq2(t *testing.T) {
	t.Run("yields zero value and error", func(t *testing.T) {
		err := errors.New("test error")
		var count int
		for val, e := range IterSeq2[string](err) {
			assert.Equal(t, "", val)
			assert.Equal(t, err, e)
			count++
		}
		assert.Equal(t, 1, count)
	})

	t.Run("yields zero int and error", func(t *testing.T) {
		err := errors.New("test error")
		for val, e := range IterSeq2[int](err) {
			assert.Equal(t, 0, val)
			assert.Equal(t, err, e)
		}
	})

	t.Run("yields zero struct and error", func(t *testing.T) {
		type result struct{ ID int }
		err := errors.New("test error")
		for val, e := range IterSeq2[result](err) {
			assert.Equal(t, result{}, val)
			assert.Equal(t, err, e)
		}
	})
}
