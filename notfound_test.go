package errs

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
)

func ExampleErrNotFound() {
	var ErrUserNotFound = fmt.Errorf("user %w", ErrNotFound)
	fmt.Println(`IsErrNotFound(ErrUserNotFound):`, IsErrNotFound(ErrUserNotFound))

	fmt.Println(`IsErrNotFound(ErrNotFound):`, IsErrNotFound(ErrNotFound))
	fmt.Println(`IsErrNotFound(sql.ErrNoRows):`, IsErrNotFound(sql.ErrNoRows))
	fmt.Println(`IsErrNotFound(os.ErrNotExist):`, IsErrNotFound(os.ErrNotExist))
	fmt.Println(`IsErrNotFound(Errorf("resource %w", ErrNotFound)):`, IsErrNotFound(Errorf("resource %w", ErrNotFound)))

	fmt.Println()
	fmt.Println(`IsErrNotFound(nil):`, IsErrNotFound(nil))
	fmt.Println(`errors.Is(sql.ErrNoRows, ErrNotFound):`, errors.Is(sql.ErrNoRows, ErrNotFound))
	fmt.Println(`errors.Is(os.ErrNotExist, ErrNotFound):`, errors.Is(os.ErrNotExist, ErrNotFound))

	// Output:
	// IsErrNotFound(ErrUserNotFound): true
	// IsErrNotFound(ErrNotFound): true
	// IsErrNotFound(sql.ErrNoRows): true
	// IsErrNotFound(os.ErrNotExist): true
	// IsErrNotFound(Errorf("resource %w", ErrNotFound)): true
	//
	// IsErrNotFound(nil): false
	// errors.Is(sql.ErrNoRows, ErrNotFound): false
	// errors.Is(os.ErrNotExist, ErrNotFound): false
}
