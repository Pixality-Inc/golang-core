package storage_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/pixality-inc/golang-core/storage"
	"github.com/pixality-inc/golang-core/storage/providers"
	"github.com/stretchr/testify/require"
)

func newOsStorage(t *testing.T) (storage.Storage, string) {
	t.Helper()

	root := t.TempDir()

	return storage.NewStorage(providers.NewOsProvider(root), providers.NewNoUrlProvider("")), root
}

type copyFallbackStorage struct {
	files map[string]string

	nativeCopyErr error
	readErr       error
	writeErr      error

	nativeCopyCalls int
	readFileCalls   int
	writeCalls      int
	sourceClosed    bool
}

type trackedReadCloser struct {
	*bytes.Reader
	closed *bool
}

func (r *trackedReadCloser) Close() error {
	*r.closed = true
	return nil
}

func (s *copyFallbackStorage) FileExists(_ context.Context, path string) (bool, error) {
	_, ok := s.files[path]
	return ok, nil
}

func (s *copyFallbackStorage) DeleteFile(_ context.Context, path string) error {
	delete(s.files, path)
	return nil
}

func (s *copyFallbackStorage) DeleteDir(_ context.Context, _ string) error {
	return nil
}

func (s *copyFallbackStorage) Write(_ context.Context, path string, file io.Reader) error {
	s.writeCalls++
	if s.writeErr != nil {
		return s.writeErr
	}

	payload, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	if s.files == nil {
		s.files = map[string]string{}
	}
	s.files[path] = string(payload)

	return nil
}

func (s *copyFallbackStorage) WriteFile(context.Context, string, string) error {
	return nil
}

func (s *copyFallbackStorage) ReadFile(_ context.Context, path string) (io.ReadCloser, error) {
	s.readFileCalls++
	if s.readErr != nil {
		return nil, s.readErr
	}

	payload, ok := s.files[path]
	if !ok {
		return nil, os.ErrNotExist
	}

	return &trackedReadCloser{
		Reader: bytes.NewReader([]byte(payload)),
		closed: &s.sourceClosed,
	}, nil
}

func (s *copyFallbackStorage) DownloadFile(context.Context, string, string) error {
	return nil
}

func (s *copyFallbackStorage) ReadDir(context.Context, string) ([]storage.DirEntry, error) {
	return nil, nil
}

func (s *copyFallbackStorage) MkDir(context.Context, string) error {
	return nil
}

func (s *copyFallbackStorage) Copy(_ context.Context, srcPath string, dstPath string) error {
	s.nativeCopyCalls++
	if s.nativeCopyErr != nil {
		return s.nativeCopyErr
	}

	payload, ok := s.files[srcPath]
	if !ok {
		return os.ErrNotExist
	}

	if s.files == nil {
		s.files = map[string]string{}
	}
	s.files[dstPath] = payload

	return nil
}

func (s *copyFallbackStorage) Move(context.Context, string, string) error {
	return nil
}

func (s *copyFallbackStorage) CreateMultipartUpload(context.Context, string) (storage.MultipartUpload, error) {
	return nil, nil
}

func (s *copyFallbackStorage) UploadMultipartChunk(context.Context, string, storage.MultipartUpload, int, io.Reader, int64) (storage.MultipartChunk, error) {
	return nil, nil
}

func (s *copyFallbackStorage) CompleteMultipartUpload(context.Context, string, storage.MultipartUpload, []storage.MultipartChunk) error {
	return nil
}

func (s *copyFallbackStorage) AbortMultipartUpload(context.Context, string, storage.MultipartUpload) error {
	return nil
}

func (s *copyFallbackStorage) GetPublicUrl(context.Context, string) (string, error) {
	return "", nil
}

func (s *copyFallbackStorage) Close() error {
	return nil
}

// TestCopy_SameStorage_nativeCopy verifies the dst==src shortcut performs a real
// server-side copy via the provider (duplicates bytes, keeps the source).
func TestCopy_SameStorage_nativeCopy(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store, root := newOsStorage(t)

	want := []byte("same-storage")
	require.NoError(t, store.Write(ctx, "a/src.bin", bytes.NewReader(want)))

	require.NoError(t, storage.Copy(ctx, store, "b/dst.bin", store, "a/src.bin"))

	got, err := os.ReadFile(filepath.Join(root, "b/dst.bin"))
	require.NoError(t, err)
	require.Equal(t, want, got)

	exists, err := store.FileExists(ctx, "a/src.bin")
	require.NoError(t, err)
	require.True(t, exists)
}

func TestCopySameStorageReturnsAfterSuccessfulNativeCopy(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := &copyFallbackStorage{
		files: map[string]string{"src": "payload"},
	}

	require.NoError(t, storage.Copy(ctx, store, "dst", store, "src"))

	require.Equal(t, 1, store.nativeCopyCalls)
	require.Zero(t, store.readFileCalls)
	require.Zero(t, store.writeCalls)
	require.Equal(t, "payload", store.files["dst"])
	require.False(t, store.sourceClosed)
}

func TestCopySameStorageFallsBackToStreamingWhenNativeCopyFails(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store := &copyFallbackStorage{
		files:         map[string]string{"src": "payload"},
		nativeCopyErr: errors.New("access denied"),
	}

	require.NoError(t, storage.Copy(ctx, store, "dst", store, "src"))

	require.Equal(t, 1, store.nativeCopyCalls)
	require.Equal(t, 1, store.readFileCalls)
	require.Equal(t, 1, store.writeCalls)
	require.Equal(t, "payload", store.files["dst"])
	require.True(t, store.sourceClosed)
}

func TestCopyStreamingFallbackReturnsReadFileError(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	readErr := errors.New("read source")
	store := &copyFallbackStorage{
		files:         map[string]string{"src": "payload"},
		nativeCopyErr: errors.New("access denied"),
		readErr:       readErr,
	}

	err := storage.Copy(ctx, store, "dst", store, "src")

	require.ErrorIs(t, err, readErr)
	require.Equal(t, 1, store.nativeCopyCalls)
	require.Equal(t, 1, store.readFileCalls)
	require.Zero(t, store.writeCalls)
	require.False(t, store.sourceClosed)
}

func TestCopyStreamingFallbackReturnsWriteErrorAndClosesSource(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	writeErr := errors.New("write destination")
	store := &copyFallbackStorage{
		files:         map[string]string{"src": "payload"},
		nativeCopyErr: errors.New("access denied"),
		writeErr:      writeErr,
	}

	err := storage.Copy(ctx, store, "dst", store, "src")

	require.ErrorIs(t, err, writeErr)
	require.Equal(t, 1, store.nativeCopyCalls)
	require.Equal(t, 1, store.readFileCalls)
	require.Equal(t, 1, store.writeCalls)
	require.True(t, store.sourceClosed)
}

// TestCopy_DifferentStorages_streams verifies that across two distinct storages
// the helper falls back to streaming (ReadFile -> Write) and content matches.
func TestCopy_DifferentStorages_streams(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	src, _ := newOsStorage(t)
	dst, dstRoot := newOsStorage(t)

	want := []byte("cross-storage")
	require.NoError(t, src.Write(ctx, "src.bin", bytes.NewReader(want)))

	require.NoError(t, storage.Copy(ctx, dst, "dst.bin", src, "src.bin"))

	got, err := os.ReadFile(filepath.Join(dstRoot, "dst.bin"))
	require.NoError(t, err)
	require.Equal(t, want, got)
}

// TestMove_SameStorage_nativeMove verifies the dst==src shortcut moves via the provider.
func TestMove_SameStorage_nativeMove(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	store, root := newOsStorage(t)

	want := []byte("move-me")
	require.NoError(t, store.Write(ctx, "src.bin", bytes.NewReader(want)))

	require.NoError(t, storage.Move(ctx, store, "dst.bin", store, "src.bin"))

	got, err := os.ReadFile(filepath.Join(root, "dst.bin"))
	require.NoError(t, err)
	require.Equal(t, want, got)

	exists, err := store.FileExists(ctx, "src.bin")
	require.NoError(t, err)
	require.False(t, exists)
}
