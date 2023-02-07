package errs

import (
	"go/build"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTrimFilePathPrefix(t *testing.T) {
	goPath := build.Default.GOPATH
	require.NotEmpty(t, goPath, "GOPATH")
	// $GOPATH/src/
	expected := filepath.Clean(goPath) + string(filepath.Separator) + "src" + string(filepath.Separator)
	require.Equal(t, expected, TrimFilePathPrefix, "TrimFilePathPrefix")
}
