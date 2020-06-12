package errs

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func Test_Combine(t *testing.T) {
	const (
		e0 = Sentinel("e0")
		e1 = Sentinel("e1")
		e2 = Sentinel("e2")
	)

	err := Combine()
	assert.NoError(t, err)

	err = Combine(nil)
	assert.NoError(t, err)

	err = Combine(nil, nil)
	assert.NoError(t, err)

	err = Combine(e0)
	assert.EqualError(t, err, "e0")
	assert.True(t, errors.Is(err, e0), "combined error is e0")

	err = Combine(nil, e0)
	assert.EqualError(t, err, "e0")
	assert.True(t, errors.Is(err, e0), "combined error is e0")

	err = Combine(e0, nil)
	assert.EqualError(t, err, "e0")
	assert.True(t, errors.Is(err, e0), "combined error is e0")

	err = Combine(e0, e1)
	assert.EqualError(t, err, "e0\ne1")
	assert.True(t, errors.Is(err, e0), "combined error is e0")
	assert.True(t, errors.Is(err, e1), "combined error is e1")

	err = Combine(e0, e1, nil)
	assert.EqualError(t, err, "e0\ne1")
	assert.True(t, errors.Is(err, e0), "combined error is e0")
	assert.True(t, errors.Is(err, e1), "combined error is e1")

	err = Combine(nil, e0, e1, nil)
	assert.EqualError(t, err, "e0\ne1")
	assert.True(t, errors.Is(err, e0), "combined error is e0")
	assert.True(t, errors.Is(err, e1), "combined error is e1")

	err = Combine(nil, e0, nil, e1, nil)
	assert.EqualError(t, err, "e0\ne1")
	assert.True(t, errors.Is(err, e0), "combined error is e0")
	assert.True(t, errors.Is(err, e1), "combined error is e1")

	err = Combine(e0, e1, e2)
	assert.EqualError(t, err, "e0\ne1\ne2")

	err = Combine(e0, nil, e2)
	assert.EqualError(t, err, "e0\ne2")

	err = Combine(e0, Combine(e1, e2))
	assert.EqualError(t, err, "e0\ne1\ne2")

	var sentErr Sentinel
	assert.True(t, errors.As(err, &sentErr), "combined error as Sentinel")
	assert.EqualError(t, sentErr, "e0", "first error e0 found as Sentinel")
}

func Test_Uncombine(t *testing.T) {
	const (
		e0 = Sentinel("e0")
		e1 = Sentinel("e1")
		e2 = Sentinel("e2")
	)

	err := Combine(e0, e1, e2)
	assert.EqualError(t, err, "e0\ne1\ne2")

	errs := Uncombine(err)
	assert.Len(t, errs, 3)
	assert.Equal(t, e0, errs[0])
	assert.Equal(t, e1, errs[1])
	assert.Equal(t, e2, errs[2])
}
