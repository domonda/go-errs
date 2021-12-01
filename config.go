package errs

import (
	"runtime"
	"strings"
)

func srcPath() string {
	// This Go package is hosted on GitHub
	// so there should always be "github.com"
	// in the path of this source file
	// if it was cloned using standard go get
	_, file, _, _ := runtime.Caller(1)
	return file[:strings.Index(file, "github.com")]
}

// TrimFilePathPrefix will be trimmed from the
// beginning of every call-stack file-path.
// Defaults to $GOPATH/src/ of the build environment.
var TrimFilePathPrefix = srcPath()
