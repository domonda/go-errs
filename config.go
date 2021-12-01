package errs

import (
	"runtime"
	"strings"
)

func staticBuildTimeGOPATH() string {
	// This Go package is hosted on GitHub
	// so there should always be "github.com"
	// in the path of this source file
	// if it was cloned using standard go get
	_, file, _, _ := runtime.Caller(1)
	end := strings.Index(file, "github.com")
	if end == -1 {
		panic("expected github.com in call-stack file-path, but got: " + file)
	}
	goPath := file[:end]
	switch {
	case strings.HasSuffix(goPath, "/pkg/mod/"):
		goPath = strings.TrimSuffix(goPath, "pkg/mod/")
	case strings.HasSuffix(goPath, "/src/"):
		goPath = strings.TrimSuffix(goPath, "src/")
	default:
		panic("expected /pkg/mod/ or /src/ in call-stack file-path, but got: " + file)
	}
	return goPath
}

// TrimFilePathPrefix will be trimmed from the
// beginning of every call-stack file-path.
// Defaults to $GOPATH of the build environment.
var TrimFilePathPrefix = staticBuildTimeGOPATH()
