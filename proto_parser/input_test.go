package proto_parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFileInput(t *testing.T) {
	t.Parallel()

	testFile, err := os.CreateTemp(t.TempDir(), "1.proto")
	require.NoError(t, err)

	_, err = testFile.WriteString("Hello")
	require.NoError(t, err)

	t.Cleanup(func() {
		fErr := testFile.Close()
		require.NoError(t, fErr)
	})

	fileInput := NewFileInput(testFile.Name(), "example")

	require.Equal(t, filepath.Base(testFile.Name()), fileInput.Name())
	require.Equal(t, "example", fileInput.Package())

	source, err := fileInput.Source()
	require.NoError(t, err)

	fErr := source.Close()
	require.NoError(t, fErr)
}
