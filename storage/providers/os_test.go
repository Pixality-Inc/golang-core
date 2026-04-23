package providers

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
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
	require.NoError(t, store.Write(ctx, "sub/file.bin", strings.NewReader(string(want))))

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

func TestOsProvider_Compose_noChunks(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	store := NewOsProvider(root)

	err := store.Compose(ctx, "out.bin", nil)
	require.ErrorIs(t, err, ErrNoChunksProvided)

	err = store.Compose(ctx, "out.bin", []string{})
	require.ErrorIs(t, err, ErrNoChunksProvided)
}

func TestOsProvider_Compose_singleChunk_renames(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	store := NewOsProvider(root)

	require.NoError(t, os.WriteFile(filepath.Join(root, "chunk0"), []byte("only"), 0o600))

	require.NoError(t, store.Compose(ctx, "final/out.txt", []string{"chunk0"}))

	b, err := os.ReadFile(filepath.Join(root, "final", "out.txt"))
	require.NoError(t, err)
	require.Equal(t, "only", string(b))

	_, err = os.Stat(filepath.Join(root, "chunk0"))
	require.True(t, os.IsNotExist(err))
}

func TestOsProvider_Compose_multipleChunks_concatenates(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	store := NewOsProvider(root)

	require.NoError(t, os.WriteFile(filepath.Join(root, "c1"), []byte("aa"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "c2"), []byte("bb"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(root, "c3"), []byte("cc"), 0o600))

	require.NoError(t, store.Compose(ctx, "merged.txt", []string{"c1", "c2", "c3"}))

	b, err := os.ReadFile(filepath.Join(root, "merged.txt"))
	require.NoError(t, err)
	require.Equal(t, "aabbcc", string(b))
}

func TestOsProvider_Compose_chunkMissing_wrapsErrChunkProcess(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	store := NewOsProvider(root)

	require.NoError(t, os.WriteFile(filepath.Join(root, "ok"), []byte("x"), 0o600))

	err := store.Compose(ctx, "out.txt", []string{"ok", "missing"})
	require.Error(t, err)
	require.ErrorIs(t, err, ErrChunkProcess)
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
