package providers

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pixality-inc/golang-core/storage"
	"github.com/stretchr/testify/require"
)

func TestOsProviderStat(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	root := t.TempDir()
	dir := filepath.Join(root, "nested")
	filename := filepath.Join(dir, "object.bin")
	require.NoError(t, os.MkdirAll(dir, 0o700))
	require.NoError(t, os.WriteFile(filename, []byte("metadata"), 0o600))

	wantModTime := time.Unix(1700000000, 0)
	require.NoError(t, os.Chtimes(filename, wantModTime, wantModTime))

	store := storage.NewLocalStorage(NewOsProvider(root), nil)
	info, err := store.Stat(ctx, "nested/object.bin")
	require.NoError(t, err)
	require.Equal(t, "object.bin", info.Name())
	require.Equal(t, int64(8), info.Size())
	require.Equal(t, fs.FileMode(0o600), info.Mode())
	require.Equal(t, wantModTime, info.ModTime())
	require.False(t, info.IsDir())

	info, err = store.Stat(ctx, "nested")
	require.NoError(t, err)
	require.Equal(t, "nested", info.Name())
	require.True(t, info.IsDir())

	info, err = store.Stat(ctx, "missing.bin")
	require.Nil(t, info)
	require.ErrorIs(t, err, fs.ErrNotExist)
	require.True(t, storage.IsNotFound(err))
}

type statStorage struct {
	storage.Storage

	info  fs.FileInfo
	err   error
	calls int
}

func (s *statStorage) Stat(context.Context, string) (fs.FileInfo, error) {
	s.calls++

	return s.info, s.err
}

func TestSyncStat(t *testing.T) {
	t.Parallel()

	want, err := storage.NewFileEntry("object.bin", 42, time.Unix(1700000000, 0)).Info()
	require.NoError(t, err)

	t.Run("first available storage", func(t *testing.T) {
		t.Parallel()

		missing := &statStorage{err: fs.ErrNotExist}
		available := &statStorage{info: want}
		unused := &statStorage{err: fs.ErrPermission}
		store := NewSync(missing, available, unused)

		info, err := store.Stat(context.Background(), "nested/object.bin")
		require.NoError(t, err)
		require.Same(t, want, info)
		require.Equal(t, 1, missing.calls)
		require.Equal(t, 1, available.calls)
		require.Zero(t, unused.calls)
	})

	t.Run("all storages fail", func(t *testing.T) {
		t.Parallel()

		store := NewSync(&statStorage{err: fs.ErrNotExist}, &statStorage{err: fs.ErrPermission})

		info, err := store.Stat(context.Background(), "nested/object.bin")
		require.Nil(t, info)
		require.ErrorIs(t, err, ErrStorageFailed)
		require.ErrorIs(t, err, fs.ErrNotExist)
		require.ErrorIs(t, err, fs.ErrPermission)
		require.True(t, storage.IsNotFound(err))
	})

	t.Run("no storages", func(t *testing.T) {
		t.Parallel()

		info, err := NewSync().Stat(context.Background(), "object.bin")
		require.Nil(t, info)
		require.ErrorIs(t, err, ErrStorageFailed)
	})
}
