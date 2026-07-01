package errs

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTrimFilePathPrefix_DefaultEmpty(t *testing.T) {
	// Empty by default: call-stack file-paths are shown in the
	// checkout-independent import-path form (see callStackFilePath), so the
	// output no longer depends on where the module is checked out.
	require.Equal(t, "", TrimFilePathPrefix, "TrimFilePathPrefix default")
}
