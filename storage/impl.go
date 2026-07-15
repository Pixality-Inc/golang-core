package storage

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/pixality-inc/golang-core/logger"
	"github.com/pixality-inc/golang-core/retry"
)

type Impl struct {
	log         logger.Loggable
	provider    Provider
	urlProvider UrlProvider
	retryPolicy retry.Policy // nil => no retry (default)
	retryable   Retryable    // nil => defaultRetryable
}

func NewStorage(provider Provider, urlProvider UrlProvider, opts ...Option) Storage {
	impl := &Impl{
		log:         logger.NewLoggableImplWithService("storage"),
		provider:    provider,
		urlProvider: urlProvider,
	}

	for _, opt := range opts {
		opt(impl)
	}

	return impl
}

func (s *Impl) FileExists(ctx context.Context, path string) (bool, error) {
	var result bool

	err := s.withRetry(ctx, func() error {
		r, err := s.provider.FileExists(ctx, path)
		if err != nil {
			return err
		}

		result = r

		return nil
	})
	if err != nil {
		return false, fmt.Errorf("storage.FileExists(%s): %w", path, err)
	}

	return result, nil
}

func (s *Impl) DeleteFile(ctx context.Context, path string) error {
	if err := s.withRetry(ctx, func() error { return s.provider.DeleteFile(ctx, path) }); err != nil {
		return fmt.Errorf("storage.DeleteFile(%s): %w", path, err)
	}

	return nil
}

func (s *Impl) DeleteDir(ctx context.Context, path string) error {
	if err := s.withRetry(ctx, func() error { return s.provider.DeleteDir(ctx, path) }); err != nil {
		return fmt.Errorf("storage.DeleteDir(%s): %w", path, err)
	}

	return nil
}

// Write is intentionally NOT retried at this layer: file is a single-use
// io.Reader, so re-invoking after a partial read would corrupt the object.
// Streaming-upload retry belongs in the provider (byte-offset resume), e.g.
// gcs.WithUploadRetry.
func (s *Impl) Write(ctx context.Context, path string, file io.Reader) error {
	if err := s.provider.Write(ctx, path, file); err != nil {
		return fmt.Errorf("storage.Write(%s): %w", path, err)
	}

	return nil
}

// WriteFile re-opens filename on every attempt, so unlike the streaming Write it
// is safe to retry: each attempt gets a fresh reader positioned at the start.
func (s *Impl) WriteFile(ctx context.Context, path string, filename string) error {
	err := s.withRetry(ctx, func() error {
		file, err := os.Open(filename)
		if err != nil {
			return fmt.Errorf("could not open file '%s': %w", filename, err)
		}

		defer func() {
			if fErr := file.Close(); fErr != nil {
				s.log.GetLogger(ctx).WithError(fErr).Errorf("failed to close file '%s'", filename)
			}
		}()

		return s.provider.Write(ctx, path, file)
	})
	if err != nil {
		return fmt.Errorf("storage.WriteFile(%s, %s): %w", path, filename, err)
	}

	return nil
}

// ReadFile retries only the stream open; reads from the returned stream are the
// caller's and cannot be retried here.
func (s *Impl) ReadFile(ctx context.Context, path string) (io.ReadCloser, error) {
	var file io.ReadCloser

	err := s.withRetry(ctx, func() error {
		reader, err := s.provider.ReadFile(ctx, path)
		if err != nil {
			// guard the (rc != nil, err != nil) contract violation: without this a
			// retried open would leak a reader per attempt.
			if reader != nil {
				_ = reader.Close()
			}

			return err
		}

		file = reader

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("storage.ReadFile(%s): %w", path, err)
	}

	return file, nil
}

// DownloadFile re-creates (truncates) the destination on every attempt, so a
// retried download restarts cleanly rather than appending to a partial file.
func (s *Impl) DownloadFile(ctx context.Context, path string, filename string) error {
	err := s.withRetry(ctx, func() error {
		file, err := s.provider.ReadFile(ctx, path)
		if err != nil {
			return err
		}

		defer func() {
			if fErr := file.Close(); fErr != nil {
				s.log.GetLogger(ctx).WithError(fErr).Errorf("failed to close file '%s'", path)
			}
		}()

		destFile, err := os.Create(filename)
		if err != nil {
			return fmt.Errorf("failed to open file %s: %w", filename, err)
		}

		if _, err := io.Copy(destFile, file); err != nil {
			_ = destFile.Close()

			return fmt.Errorf("failed to copy file %s to %s: %w", path, filename, err)
		}

		// return the destination close error rather than only logging it: a failed
		// flush (e.g. ENOSPC) leaves a truncated file, and surfacing it lets the
		// retry re-create and re-download instead of reporting a false success.
		if err := destFile.Close(); err != nil {
			return fmt.Errorf("failed to close file %s: %w", filename, err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("storage.DownloadFile(%s): %w", path, err)
	}

	return nil
}

func (s *Impl) ReadDir(ctx context.Context, path string) ([]DirEntry, error) {
	var dirEntries []DirEntry

	err := s.withRetry(ctx, func() error {
		entries, err := s.provider.ReadDir(ctx, path)
		if err != nil {
			return err
		}

		dirEntries = entries

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("storage.ReadDir(%s): %w", path, err)
	}

	return dirEntries, nil
}

func (s *Impl) MkDir(ctx context.Context, path string) error {
	if err := s.withRetry(ctx, func() error { return s.provider.MkDir(ctx, path) }); err != nil {
		return fmt.Errorf("storage.MkDir(%s): %w", path, err)
	}

	return nil
}

func (s *Impl) Copy(ctx context.Context, srcPath string, dstPath string) error {
	if err := s.withRetry(ctx, func() error { return s.provider.Copy(ctx, srcPath, dstPath) }); err != nil {
		return fmt.Errorf("storage.Copy(%s -> %s): %w", srcPath, dstPath, err)
	}

	return nil
}

func (s *Impl) Move(ctx context.Context, srcPath string, dstPath string) error {
	if err := s.withRetry(ctx, func() error { return s.provider.Move(ctx, srcPath, dstPath) }); err != nil {
		return fmt.Errorf("storage.Move(%s -> %s): %w", srcPath, dstPath, err)
	}

	return nil
}

func (s *Impl) CreateMultipartUpload(ctx context.Context, path string) (MultipartUpload, error) {
	upload, err := s.provider.CreateMultipartUpload(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("storage.CreateMultipartUpload(%s): %w", path, err)
	}

	return upload, nil
}

func (s *Impl) UploadMultipartChunk(ctx context.Context, path string, upload MultipartUpload, chunkNumber int, body io.Reader, size int64) (MultipartChunk, error) {
	chunk, err := s.provider.UploadMultipartChunk(ctx, path, upload, chunkNumber, body, size)
	if err != nil {
		return nil, fmt.Errorf("storage.UploadMultipartChunk(%s, %s, %d): %w", path, upload.Id(), chunkNumber, err)
	}

	return chunk, nil
}

func (s *Impl) CompleteMultipartUpload(ctx context.Context, path string, upload MultipartUpload, chunks []MultipartChunk) error {
	if err := s.provider.CompleteMultipartUpload(ctx, path, upload, chunks); err != nil {
		return fmt.Errorf("storage.CompleteMultipartUpload(%s, %s, %d chunks): %w", path, upload.Id(), len(chunks), err)
	}

	return nil
}

func (s *Impl) AbortMultipartUpload(ctx context.Context, path string, upload MultipartUpload) error {
	if err := s.provider.AbortMultipartUpload(ctx, path, upload); err != nil {
		return fmt.Errorf("storage.AbortMultipartUpload(%s, %s): %w", path, upload.Id(), err)
	}

	return nil
}

func (s *Impl) Close() error {
	if err := s.provider.Close(); err != nil {
		return fmt.Errorf("storage.Close(): %w", err)
	}

	return nil
}

func (s *Impl) GetPublicUrl(ctx context.Context, path string) (string, error) {
	var url string

	err := s.withRetry(ctx, func() error {
		u, err := s.urlProvider.GetPublicUrl(ctx, path)
		if err != nil {
			return err
		}

		url = u

		return nil
	})
	if err != nil {
		return "", fmt.Errorf("storage.GetPublicUrl(%s): %w", path, err)
	}

	return url, nil
}
