package errs

import "errors"

// Logger is an interface that can be implemented to log errors
type Logger interface {
	Printf(format string, args ...interface{})
}

type LogDecisionMaker interface {
	ShouldLog() bool
}

func ShouldLog(err error) bool {
	var logDecisionMaker LogDecisionMaker
	return errors.As(err, &logDecisionMaker) && logDecisionMaker.ShouldLog()
}
