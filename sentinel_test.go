package errs

import (
	"errors"
	"fmt"
)

func ExampleSentinel() {
	const ErrUserNotFound Sentinel = "user not found"

	var err error = ErrUserNotFound
	fmt.Println("const Sentinel errors.Is:", errors.Is(err, ErrUserNotFound))

	err = fmt.Errorf("%w: user@example.com", ErrUserNotFound)
	fmt.Println("Wrapped Sentinel errors.Is:", errors.Is(err, ErrUserNotFound))

	// Output:
	// const Sentinel errors.Is: true
	// Wrapped Sentinel errors.Is: true
}
