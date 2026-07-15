package storage

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pixality-inc/golang-core/retry"
	"github.com/stretchr/testify/require"
)

var (
	errConnReset       = errors.New("connection reset")
	errTransient error = &net.OpError{Op: "read", Err: errConnReset} // net.Error => retryable by defaultRetryable
	errPermanent       = errors.New("permanent")                     // generic  => not retryable by defaultRetryable
)

// fakeProvider embeds the Provider interface (nil) so only the methods a test
// actually calls need to be overridden; any other call panics, which is what we
// want: it flags an unexpected code path.
type fakeProvider struct {
	Provider

	fileExistsErrs []error // per-call error; index past the slice => success
	fileExistsN    int

	writeErrs  []error  // per-call error; index past the slice => success
	writeReads []string // bytes seen by each Write call, in order

	readFileErrs []error // per-call error; index past the slice => success
	readFileN    int
	readContent  string // content returned by ReadFile on success
}

func (f *fakeProvider) FileExists(_ context.Context, _ string) (bool, error) {
	i := f.fileExistsN
	f.fileExistsN++

	if i < len(f.fileExistsErrs) && f.fileExistsErrs[i] != nil {
		return false, f.fileExistsErrs[i]
	}

	return true, nil
}

func (f *fakeProvider) Write(_ context.Context, _ string, r io.Reader) error {
	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}

	f.writeReads = append(f.writeReads, string(b))

	i := len(f.writeReads) - 1
	if i < len(f.writeErrs) {
		return f.writeErrs[i]
	}

	return nil
}

func (f *fakeProvider) ReadFile(_ context.Context, _ string) (io.ReadCloser, error) {
	i := f.readFileN
	f.readFileN++

	if i < len(f.readFileErrs) && f.readFileErrs[i] != nil {
		return nil, f.readFileErrs[i]
	}

	return io.NopCloser(strings.NewReader(f.readContent)), nil
}

func enabledPolicy() retry.Policy {
	return retry.NewPolicy(
		retry.WithEnabled(true),
		retry.WithMaxAttempts(3),
		retry.WithInitialInterval(time.Millisecond),
	)
}

func TestWithRetry_RetriesTransientOp(t *testing.T) {
	t.Parallel()

	provider := &fakeProvider{fileExistsErrs: []error{errTransient, nil}}
	s := NewStorage(provider, nil, WithRetry(enabledPolicy()))

	ok, err := s.FileExists(context.Background(), "some/path")
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, 2, provider.fileExistsN, "transient error should retry once then succeed")
}

func TestWithRetry_DisabledByDefault(t *testing.T) {
	t.Parallel()

	provider := &fakeProvider{fileExistsErrs: []error{errTransient, nil}}
	s := NewStorage(provider, nil) // no WithRetry

	_, err := s.FileExists(context.Background(), "some/path")
	require.Error(t, err)
	require.Equal(t, 1, provider.fileExistsN, "without WithRetry the op runs exactly once")
}

// TestWithRetry_DefaultSkipsPermanent documents the narrowed default classifier:
// a non-transport error is not retried, so an idempotent op fails fast instead of
// burning its whole attempt budget on a doomed call.
func TestWithRetry_DefaultSkipsPermanent(t *testing.T) {
	t.Parallel()

	provider := &fakeProvider{fileExistsErrs: []error{errPermanent, nil}}
	s := NewStorage(provider, nil, WithRetry(enabledPolicy()))

	_, err := s.FileExists(context.Background(), "some/path")
	require.Error(t, err)
	require.Equal(t, 1, provider.fileExistsN, "permanent error must not be retried by default")
}

// TestWithRetryClassifier_Overrides shows a custom classifier fully controls what
// is retried: here it refuses everything, so even a normally-transient error
// fails on the first attempt.
func TestWithRetryClassifier_Overrides(t *testing.T) {
	t.Parallel()

	provider := &fakeProvider{fileExistsErrs: []error{errTransient, nil}}
	s := NewStorage(provider, nil,
		WithRetry(enabledPolicy()),
		WithRetryClassifier(func(error) bool { return false }),
	)

	_, err := s.FileExists(context.Background(), "some/path")
	require.Error(t, err)
	require.Equal(t, 1, provider.fileExistsN, "classifier returning false must stop retries")
}

// TestWithRetry_WriteFileReopensPerAttempt is the key safety test: a retried
// WriteFile must re-open the source file so every attempt sends the full content
// from offset 0 (never a drained reader).
func TestWithRetry_WriteFileReopensPerAttempt(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	fname := filepath.Join(dir, "src.txt")
	require.NoError(t, os.WriteFile(fname, []byte("hello"), 0o600))

	provider := &fakeProvider{writeErrs: []error{errTransient}} // fail once, then succeed
	s := NewStorage(provider, nil, WithRetry(enabledPolicy()))

	err := s.WriteFile(context.Background(), "dst", fname)
	require.NoError(t, err)
	require.Equal(t, []string{"hello", "hello"}, provider.writeReads,
		"each attempt must re-read the full file from the start")
}

// TestWithRetry_StreamingWriteNotRetried asserts the streaming Write is excluded
// from storage-level retry even when a policy is set: a single-use reader must
// not be re-sent from this layer.
func TestWithRetry_StreamingWriteNotRetried(t *testing.T) {
	t.Parallel()

	provider := &fakeProvider{writeErrs: []error{errTransient}}
	s := NewStorage(provider, nil, WithRetry(enabledPolicy()))

	err := s.Write(context.Background(), "dst", bytes.NewReader([]byte("x")))
	require.Error(t, err)
	require.Len(t, provider.writeReads, 1, "streaming Write must run exactly once")
}

// TestWithRetry_ReadFileRetriesOpen covers the ReadFile open-retry path.
func TestWithRetry_ReadFileRetriesOpen(t *testing.T) {
	t.Parallel()

	provider := &fakeProvider{readFileErrs: []error{errTransient}, readContent: "payload"}
	s := NewStorage(provider, nil, WithRetry(enabledPolicy()))

	reader, err := s.ReadFile(context.Background(), "some/path")
	require.NoError(t, err)
	require.Equal(t, 2, provider.readFileN, "transient open error should retry once")

	content, err := io.ReadAll(reader)
	require.NoError(t, err)
	require.NoError(t, reader.Close())
	require.Equal(t, "payload", string(content))
}

// TestWithRetry_DownloadFileRetries covers DownloadFile: a transient read-open
// error retries and the destination ends up with the full content.
func TestWithRetry_DownloadFileRetries(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	dst := filepath.Join(dir, "out.txt")

	provider := &fakeProvider{readFileErrs: []error{errTransient}, readContent: "downloaded"}
	s := NewStorage(provider, nil, WithRetry(enabledPolicy()))

	err := s.DownloadFile(context.Background(), "some/path", dst)
	require.NoError(t, err)
	require.Equal(t, 2, provider.readFileN, "transient open error should retry once")

	got, err := os.ReadFile(dst)
	require.NoError(t, err)
	require.Equal(t, "downloaded", string(got))
}
