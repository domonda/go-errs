package errs

import (
	"runtime"
	"strings"
)

func srcPath() string {
	_, file, _, _ := runtime.Caller(1)
	return file[:strings.Index(file, "github.com")]
}

// TrimFilePathPrefix will be trimmed from the
// beginning of every call-stack file-path.
// Defaults to $GOPATH/src/ of the build environment.
var TrimFilePathPrefix = srcPath()
