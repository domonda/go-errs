package errs

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTrimFilePathPrefix(t *testing.T) {
	goPath := os.Getenv("GOPATH")
	if goPath == "" {
		t.Fatal("GOPATH env var not set")
	}
	// $GOPATH/src/
	expected := filepath.Clean(goPath) + string(filepath.Separator)
	if TrimFilePathPrefix != expected {
		t.Fatalf("TrimFilePathPrefix %q is not the expected path %q", TrimFilePathPrefix, expected)
	}
}
