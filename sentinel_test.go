package errs

import (
	"errors"
	"fmt"
)

func ExampleSentinel() {
	const ErrUserAlreadyExists Sentinel = "user already exists"

	var err error = ErrUserAlreadyExists
	fmt.Println("const Sentinel errors.Is:", errors.Is(err, ErrUserAlreadyExists))

	err = fmt.Errorf("%w: user@example.com", ErrUserAlreadyExists)
	fmt.Println("Wrapped Sentinel errors.Is:", errors.Is(err, ErrUserAlreadyExists))

	// Output:
	// const Sentinel errors.Is: true
	// Wrapped Sentinel errors.Is: true
}
