package errs

import (
	"runtime"
	"strings"
)

// TrimFilePathPrefix will be trimmed from the
// beginning of every call-stack file-path.
// Defaults to $GOPATH/src/ of the build environment
// or will be empty if go build gets called with -trimpath.
var TrimFilePathPrefix = filePathPrefix()

func filePathPrefix() string {
	// This Go package is hosted on GitHub
	// so there should always be "github.com"
	// in the path of this source file
	// if it was cloned using standard go get
	_, file, _, _ := runtime.Caller(1)
	end := strings.Index(file, "github.com")
	if end == -1 {
		panic("expected github.com in call-stack file-path, but got: " + file)
	}
	return file[:end]
}
