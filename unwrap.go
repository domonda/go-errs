package errs

type unwrapper interface {
	Unwrap() error
}

// Root unwraps err recursively and returns the root error.
func Root(err error) error {
	for {
		u, ok := err.(unwrapper)
		if !ok {
			return err
		}
		e := u.Unwrap()
		if e == nil {
			return err
		}
		err = e
	}
}
