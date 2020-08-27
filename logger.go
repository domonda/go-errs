package errs

import "errors"

// Logger is an interface that can be implemented to log errors
type Logger interface {
	Printf(format string, args ...interface{})
}

// LogDecisionMaker can be implemented by errors
// to decide if they should be logged.
// Use the package function ShouldLog to check
// if a wrapped error implements the interface
// and get the result of its ShouldLog method.
type LogDecisionMaker interface {
	error

	// ShouldLog decides if the error should be logged
	ShouldLog() bool
}

// ShouldLog checks if the passed error unwraps
// as a LogDecisionMaker and returns the result
// of its ShouldLog method.
// If error does not unwrap to LogDecisionMaker
// and is not nil then ShouldLog returns true.
// If nil is passed as error, then ShouldLog returns false.
func ShouldLog(err error) bool {
	if err == nil {
		return false
	}
	var logDecisionMaker LogDecisionMaker
	if errors.As(err, &logDecisionMaker) {
		return logDecisionMaker.ShouldLog()
	}
	return true
}
