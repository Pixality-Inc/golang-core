package storage

import (
	"context"
	"errors"
	"net"

	"github.com/pixality-inc/golang-core/retry"
	"golang.org/x/net/http2"
)

// Option configures a Storage at construction time.
type Option func(*Impl)

// Retryable classifies whether a failed operation error is transient and worth
// retrying.
type Retryable func(error) bool

// WithRetry enables bounded retry for the idempotent, re-runnable Storage
// operations: the path-based ops (FileExists, DeleteFile, DeleteDir, ReadDir,
// MkDir, Copy, Move, GetPublicUrl), WriteFile (the source file is re-opened on
// every attempt), ReadFile (only the stream open is retried), and DownloadFile
// (the destination file is re-created on every attempt).
//
// it deliberately does not retry the streaming Write / UploadMultipartChunk
// paths: those consume a caller-supplied single-use io.Reader that cannot be
// re-sent safely from this layer (a re-invocation after a partial read would
// corrupt the object). retry those inside the provider, where the backend can
// resume by byte offset, e.g. gcs.WithUploadRetry.
//
// which errors count as transient is decided by defaultRetryable unless
// overridden with WithRetryClassifier.
func WithRetry(policy retry.Policy) Option {
	return func(s *Impl) {
		s.retryPolicy = policy
	}
}

// WithRetryClassifier overrides the transient-error classifier used by WithRetry.
// it has no effect unless WithRetry is also set.
func WithRetryClassifier(retryable Retryable) Option {
	return func(s *Impl) {
		s.retryable = retryable
	}
}

// defaultRetryable retries only transport-level transients: network errors and
// HTTP/2 stream resets. permanent errors (not-found, permission, invalid-arg)
// fail fast so an idempotent op does not burn its whole attempt budget on a
// doomed call. backend 5xx/429 are already retried by the underlying gcs/minio
// clients before they surface here.
func defaultRetryable(err error) bool {
	if err == nil {
		return false
	}

	// context guard first. order matters: context.DeadlineExceeded also satisfies
	// net.Error, so it must be caught before the net.Error check below.
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	var streamErr http2.StreamError

	return errors.As(err, &streamErr)
}

// withRetry runs operation under the configured retry policy. with no policy set
// it is a straight passthrough, so callers that did not opt in keep
// single-attempt behavior.
func (s *Impl) withRetry(ctx context.Context, operation func() error) error {
	if s.retryPolicy == nil {
		return operation()
	}

	retryable := s.retryable
	if retryable == nil {
		retryable = defaultRetryable
	}

	_, err := retry.DoWithCondition(
		ctx,
		s.retryPolicy,
		s.log,
		func() (struct{}, error) { return struct{}{}, operation() },
		func(_ struct{}, err error) bool { return err != nil && retryable(err) },
	)

	return err
}
