package examples

func Outer(a int) (err error) {
	inner := func(b string) (innerErr error) {
		//#wrap-result-err
		return nil
	}
	_ = inner
	return nil
}
