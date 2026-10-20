package main

import (
	"flag"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePathArgs(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		wantPath      string
		wantRecursive bool
	}{
		{name: "directory", args: []string{"./pkg"}, wantPath: "./pkg"},
		{name: "file", args: []string{"./pkg/file.go"}, wantPath: "./pkg/file.go"},
		{name: "recursive", args: []string{"./pkg/..."}, wantPath: "./pkg", wantRecursive: true},
		{name: "current directory recursive", args: []string{"./..."}, wantPath: ".", wantRecursive: true},
		{name: "bare dots", args: []string{"..."}, wantPath: ".", wantRecursive: true},
		{name: "root", args: []string{"/..."}, wantPath: "/", wantRecursive: true},
		{
			name:          "OS separator",
			args:          []string{"pkg" + string(filepath.Separator) + "..."},
			wantPath:      "pkg",
			wantRecursive: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sourcePath, recursive, err := parsePathArgs(tt.args)
			require.NoError(t, err)
			assert.Equal(t, tt.wantPath, sourcePath)
			assert.Equal(t, tt.wantRecursive, recursive)
		})
	}

	t.Run("missing path", func(t *testing.T) {
		_, _, err := parsePathArgs(nil)
		require.ErrorContains(t, err, "missing path")
	})

	// The flag package stops at the path, so a -validate after it
	// would otherwise be ignored and the files rewritten instead
	t.Run("flag after path", func(t *testing.T) {
		_, _, err := parsePathArgs([]string{"./...", "-validate"})
		require.ErrorContains(t, err, `unexpected argument "-validate"`)
	})
}

func TestCheckCommand(t *testing.T) {
	for _, command := range []string{"remove", "replace", "insert"} {
		require.NoError(t, checkCommand(command), command)
	}
	require.EqualError(t, checkCommand("rename"), `unknown command "rename"`)
	// An option before the command is not taken for the command and the
	// command for the path, which reported an argument after the path
	require.EqualError(t, checkCommand("-validate"), `unknown command "-validate", the command must come before the options`)
}

func TestCheckOutFlag(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr bool
	}{
		{name: "not passed", args: []string{"./..."}},
		{name: "path", args: []string{"-out", "./out", "./..."}},
		// Like -out "$OUTDIR" with OUTDIR unset
		{name: "empty path", args: []string{"-out", "", "./..."}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fs := flag.NewFlagSet("insert", flag.ContinueOnError)
			fs.String("out", "", "")
			err := fs.Parse(tt.args)
			require.NoError(t, err)

			err = checkOutFlag(fs)
			if tt.wantErr {
				require.ErrorContains(t, err, "-out needs a non-empty path")
			} else {
				require.NoError(t, err)
			}
		})
	}
}
