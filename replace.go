package errs

import "errors"

// ReplaceIfIs returns replacement if errors.Is(err, target) is true
// and err otherwise.
// A nil err is returned as nil and never replaced.
//
// Example:
//
//	user, err := db.GetUser(ctx, id)
//	if err != nil {
//		return errs.ReplaceIfIs(err, sql.ErrNoRows, ErrUserNotFound)
//	}
func ReplaceIfIs(err, target, replacement error) error {
	if err != nil && errors.Is(err, target) {
		return replacement
	}
	return err
}

// ReplaceIfIsNot returns replacement if errors.Is(err, target) is false
// and err otherwise.
// A nil err is returned as nil and never replaced.
//
// Example:
//
//	if err := flags.Parse(args); err != nil {
//		// Pass through flag.ErrHelp, else return errUsage
//		return errs.ReplaceIfIsNot(err, flag.ErrHelp, errUsage)
//	}
func ReplaceIfIsNot(err, target, replacement error) error {
	if err == nil || errors.Is(err, target) {
		return err
	}
	return replacement
}

// ReplaceIfType returns replacement if errors.AsType[T](err)
// finds an error of type T in err's tree and err otherwise.
// A nil err is returned as nil and never replaced.
//
// Example:
//
//	if err := json.Unmarshal(data, &config); err != nil {
//		return errs.ReplaceIfType[*json.SyntaxError](err, ErrInvalidConfig)
//	}
func ReplaceIfType[T error](err, replacement error) error {
	if _, ok := errors.AsType[T](err); ok {
		return replacement
	}
	return err
}

// ReplaceIfNotType returns replacement if errors.AsType[T](err)
// finds no error of type T in err's tree and err otherwise.
// A nil err is returned as nil and never replaced.
//
// Example:
//
//	if err := validate(input); err != nil {
//		// Pass through validation errors, else hide the details
//		return errs.ReplaceIfNotType[*ValidationError](err, ErrInternal)
//	}
func ReplaceIfNotType[T error](err, replacement error) error {
	if _, ok := errors.AsType[T](err); err == nil || ok {
		return err
	}
	return replacement
}

// ReplaceErrNotFound returns the passed replacement error
// if IsErrNotFound(err) returns true,
// meaning that all (optionally wrapped)
// ErrNotFound, sql.ErrNoRows, os.ErrNotExist
// errors get replaced.
func ReplaceErrNotFound(err, replacement error) error {
	if IsErrNotFound(err) {
		return replacement
	}
	return err
}
