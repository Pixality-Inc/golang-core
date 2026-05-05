package providers

import (
	"bytes"
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/pixality-inc/golang-core/storage"
	"github.com/stretchr/testify/require"
)

func TestOsProvider_FileExists_falseWhenMissing(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	store := NewOsProvider(root)

	exists, err := store.FileExists(ctx, "missing.txt")
	require.NoError(t, err)
	require.False(t, exists)
}

func TestOsProvider_FileExists_trueWhenPresent(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	store := NewOsProvider(root)

	require.NoError(t, os.WriteFile(filepath.Join(root, "a.txt"), []byte("x"), 0o600))

	exists, err := store.FileExists(ctx, "a.txt")
	require.NoError(t, err)
	require.True(t, exists)
}

func TestOsProvider_Write_ReadFile_roundTrip(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	store := NewOsProvider(root)

	want := []byte("hello world")
	require.NoError(t, store.Write(ctx, "sub/file.bin", bytes.NewReader(want)))

	rc, err := store.ReadFile(ctx, "sub/file.bin")
	require.NoError(t, err)
	t.Cleanup(func() { _ = rc.Close() })

	got, err := io.ReadAll(rc)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestOsProvider_ReadFile_notFound(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	store := NewOsProvider(root)

	_, err := store.ReadFile(ctx, "nope.txt")
	require.Error(t, err)
	require.ErrorIs(t, err, os.ErrNotExist)
}

func TestOsProvider_DeleteFile_removesFile(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	store := NewOsProvider(root)

	path := "to-delete.txt"
	require.NoError(t, os.WriteFile(filepath.Join(root, path), []byte("x"), 0o600))

	require.NoError(t, store.DeleteFile(ctx, path))

	_, err := os.Stat(filepath.Join(root, path))
	require.True(t, os.IsNotExist(err))
}

func TestOsProvider_DeleteFile_missingReturnsError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	store := NewOsProvider(root)

	err := store.DeleteFile(ctx, "missing.bin")
	require.Error(t, err)
}

func TestOsProvider_DeleteDir_recursive(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	store := NewOsProvider(root)

	require.NoError(t, os.MkdirAll(filepath.Join(root, "d", "nested"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(root, "d", "nested", "f.txt"), []byte("x"), 0o600))

	require.NoError(t, store.DeleteDir(ctx, "d"))

	_, err := os.Stat(filepath.Join(root, "d"))
	require.True(t, os.IsNotExist(err))
}

func TestOsProvider_MkDir_and_ReadDir(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	store := NewOsProvider(root)

	require.NoError(t, store.MkDir(ctx, "a/b"))
	require.NoError(t, os.WriteFile(filepath.Join(root, "a", "b", "one.txt"), []byte("1"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "a", "b", "two.txt"), []byte("2"), 0o600))

	entries, err := store.ReadDir(ctx, "a/b")
	require.NoError(t, err)
	require.Len(t, entries, 2)

	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}

	require.ElementsMatch(t, []string{"one.txt", "two.txt"}, names)
}

func TestOsProvider_ReadDir_notFound(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	store := NewOsProvider(root)

	_, err := store.ReadDir(ctx, "no-such-dir")
	require.Error(t, err)
}

func TestOsProvider_CompleteMultipartUpload_noChunks(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	store := NewOsProvider(root)

	err := store.CompleteMultipartUpload(ctx, "out.bin", "u1", nil)
	require.ErrorIs(t, err, ErrNoChunksProvided)

	err = store.CompleteMultipartUpload(ctx, "out.bin", "u1", []storage.MultipartChunk{})
	require.ErrorIs(t, err, ErrNoChunksProvided)
}

func TestOsProvider_Multipart_singleChunk(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	store := NewOsProvider(root)

	uploadId, err := store.CreateMultipartUpload(ctx, "final/out.txt")
	require.NoError(t, err)
	require.NotEmpty(t, uploadId)

	_, err = store.UploadMultipartChunk(ctx, "final/out.txt", uploadId, 1, bytes.NewReader([]byte("only")), 4)
	require.NoError(t, err)

	require.NoError(t, store.CompleteMultipartUpload(ctx, "final/out.txt", uploadId, []storage.MultipartChunk{
		{Number: 1},
	}))

	b, err := os.ReadFile(filepath.Join(root, "final", "out.txt"))
	require.NoError(t, err)
	require.Equal(t, "only", string(b))

	_, err = os.Stat(filepath.Join(root, "final/out.txt.parts"))
	require.True(t, os.IsNotExist(err))
}

func TestOsProvider_Multipart_multipleChunks_concatenated(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	store := NewOsProvider(root)

	uploadId, err := store.CreateMultipartUpload(ctx, "merged.txt")
	require.NoError(t, err)

	_, err = store.UploadMultipartChunk(ctx, "merged.txt", uploadId, 1, bytes.NewReader([]byte("aa")), 2)
	require.NoError(t, err)
	_, err = store.UploadMultipartChunk(ctx, "merged.txt", uploadId, 2, bytes.NewReader([]byte("bb")), 2)
	require.NoError(t, err)
	_, err = store.UploadMultipartChunk(ctx, "merged.txt", uploadId, 3, bytes.NewReader([]byte("cc")), 2)
	require.NoError(t, err)

	require.NoError(t, store.CompleteMultipartUpload(ctx, "merged.txt", uploadId, []storage.MultipartChunk{
		{Number: 1}, {Number: 2}, {Number: 3},
	}))

	b, err := os.ReadFile(filepath.Join(root, "merged.txt"))
	require.NoError(t, err)
	require.Equal(t, "aabbcc", string(b))

	_, err = os.Stat(filepath.Join(root, "merged.txt.parts"))
	require.True(t, os.IsNotExist(err))
}

func TestOsProvider_Multipart_missingChunk_wrapsErrChunkProcess(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	store := NewOsProvider(root)

	uploadId, err := store.CreateMultipartUpload(ctx, "out.txt")
	require.NoError(t, err)

	_, err = store.UploadMultipartChunk(ctx, "out.txt", uploadId, 1, bytes.NewReader([]byte("x")), 1)
	require.NoError(t, err)

	err = store.CompleteMultipartUpload(ctx, "out.txt", uploadId, []storage.MultipartChunk{
		{Number: 1}, {Number: 2},
	})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrChunkProcess)
}

func TestOsProvider_Multipart_abort_removesParts(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	store := NewOsProvider(root)

	uploadId, err := store.CreateMultipartUpload(ctx, "out.txt")
	require.NoError(t, err)

	_, err = store.UploadMultipartChunk(ctx, "out.txt", uploadId, 1, bytes.NewReader([]byte("x")), 1)
	require.NoError(t, err)

	require.NoError(t, store.AbortMultipartUpload(ctx, "out.txt", uploadId))

	_, err = os.Stat(filepath.Join(root, "out.txt.parts"))
	require.True(t, os.IsNotExist(err))
}

func TestOsProvider_LocalPath_joinsRoot(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	store := NewOsProvider(root)

	got, err := store.LocalPath(ctx, filepath.Join("store", "q.txt"))
	require.NoError(t, err)
	require.Equal(t, filepath.Join(root, "store", "q.txt"), got)
}

func TestOsProvider_Close_nilError(t *testing.T) {
	t.Parallel()

	store := NewOsProvider(t.TempDir())
	require.NoError(t, store.Close())
}

func TestOsProvider_FileExists_statErrorPropagates(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	fileRoot := filepath.Join(root, "file-as-root")
	require.NoError(t, os.WriteFile(fileRoot, []byte("x"), 0o600))

	// Root is a regular file, so joined paths are not valid for Stat (e.g. ENOTDIR).
	store := NewOsProvider(fileRoot)

	_, err := store.FileExists(ctx, "anything")
	require.Error(t, err)
}

// compile-time check that OsProvider implements storage.LocalStorageProvider.
var _ storage.LocalStorageProvider = (*OsProvider)(nil)
