package rewrite

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExamplesReplace(t *testing.T) {
	examplesDir := filepath.Join("..", "examples")
	expectedDir := filepath.Join(examplesDir, "expected")

	// Find all replace example files
	entries, err := os.ReadDir(examplesDir)
	require.NoError(t, err)

	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, "replace") ||
			entry.IsDir() ||
			!strings.HasSuffix(name, ".go") {
			continue
		}

		baseName := strings.TrimSuffix(name, ".go")
		t.Run(baseName, func(t *testing.T) {
			inputPath := filepath.Join(examplesDir, name)
			expectedPath := filepath.Join(expectedDir, name)
			outputPath := filepath.Join(examplesDir, baseName+".output.go")

			// Ensure output is cleaned up after test
			defer os.Remove(outputPath)

			// Read expected output
			expected, err := os.ReadFile(expectedPath)
			require.NoError(t, err, "expected file should exist: %s", expectedPath)

			// Run replace
			err = Replace(inputPath, outputPath, nil)
			require.NoError(t, err)

			// Read actual output
			actual, err := os.ReadFile(outputPath)
			require.NoError(t, err)

			// Compare
			assert.Equal(t, string(expected), string(actual),
				"output should match expected for %s", baseName)
		})
	}
}

func TestExamplesRemove(t *testing.T) {
	examplesDir := filepath.Join("..", "examples")
	expectedDir := filepath.Join(examplesDir, "expected")

	// Find all remove example files
	entries, err := os.ReadDir(examplesDir)
	require.NoError(t, err)

	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasPrefix(name, "remove") ||
			entry.IsDir() ||
			!strings.HasSuffix(name, ".go") {
			continue
		}

		baseName := strings.TrimSuffix(name, ".go")
		t.Run(baseName, func(t *testing.T) {
			inputPath := filepath.Join(examplesDir, name)
			expectedPath := filepath.Join(expectedDir, name)
			outputPath := filepath.Join(examplesDir, baseName+".output.go")

			// Ensure output is cleaned up after test
			defer os.Remove(outputPath)

			// Read expected output
			expected, err := os.ReadFile(expectedPath)
			require.NoError(t, err, "expected file should exist: %s", expectedPath)

			// Run remove
			err = Remove(inputPath, outputPath, nil)
			require.NoError(t, err)

			// Read actual output
			actual, err := os.ReadFile(outputPath)
			require.NoError(t, err)

			// Compare
			assert.Equal(t, string(expected), string(actual),
				"output should match expected for %s", baseName)
		})
	}
}
