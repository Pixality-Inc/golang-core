package storage

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/pixality-inc/golang-core/logger"
)

type Impl struct {
	log         logger.Loggable
	provider    Provider
	urlProvider UrlProvider
}

func NewStorage(provider Provider, urlProvider UrlProvider) Storage {
	return &Impl{
		log:         logger.NewLoggableImplWithService("storage"),
		provider:    provider,
		urlProvider: urlProvider,
	}
}

func (s *Impl) FileExists(ctx context.Context, path string) (bool, error) {
	result, err := s.provider.FileExists(ctx, path)
	if err != nil {
		return false, fmt.Errorf("storage.FileExists(%s): %w", path, err)
	}

	return result, nil
}

func (s *Impl) DeleteFile(ctx context.Context, path string) error {
	if err := s.provider.DeleteFile(ctx, path); err != nil {
		return fmt.Errorf("storage.DeleteFile(%s): %w", path, err)
	}

	return nil
}

func (s *Impl) DeleteDir(ctx context.Context, path string) error {
	if err := s.provider.DeleteDir(ctx, path); err != nil {
		return fmt.Errorf("storage.DeleteDir(%s): %w", path, err)
	}

	return nil
}

func (s *Impl) Write(ctx context.Context, path string, file io.Reader) error {
	if err := s.provider.Write(ctx, path, file); err != nil {
		return fmt.Errorf("storage.Write(%s): %w", path, err)
	}

	return nil
}

func (s *Impl) WriteFile(ctx context.Context, path string, filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return fmt.Errorf("could not open file '%s': %w", filename, err)
	}

	defer func() {
		if fErr := file.Close(); fErr != nil {
			s.log.GetLogger(ctx).WithError(err).Errorf("failed to close file '%s'", filename)
		}
	}()

	if err = s.provider.Write(ctx, path, file); err != nil {
		return fmt.Errorf("storage.WriteFile(%s, %s): %w", path, filename, err)
	}

	return nil
}

func (s *Impl) ReadFile(ctx context.Context, path string) (io.ReadCloser, error) {
	file, err := s.provider.ReadFile(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("storage.ReadFile(%s): %w", path, err)
	}

	return file, nil
}

func (s *Impl) DownloadFile(ctx context.Context, path string, filename string) error {
	file, err := s.provider.ReadFile(ctx, path)
	if err != nil {
		return fmt.Errorf("storage.DownloadFile(%s): %w", path, err)
	}

	defer func() {
		if fErr := file.Close(); fErr != nil {
			s.log.GetLogger(ctx).WithError(err).Errorf("failed to close file '%s'", path)
		}
	}()

	destFile, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("storage.DownloadFile(%s): failed to open file %s: %w", path, filename, err)
	}

	defer func() {
		if fErr := destFile.Close(); fErr != nil {
			s.log.GetLogger(ctx).WithError(err).Errorf("failed to close file '%s'", filename)
		}
	}()

	if _, err = io.Copy(destFile, file); err != nil {
		return fmt.Errorf("storage.DownloadFile(%s): failed to copy file %s to %s: %w", path, path, filename, err)
	}

	return nil
}

func (s *Impl) ReadDir(ctx context.Context, path string) ([]DirEntry, error) {
	dirEntries, err := s.provider.ReadDir(ctx, path)
	if err != nil {
		return nil, fmt.Errorf("storage.ReadDir(%s): %w", path, err)
	}

	return dirEntries, nil
}

func (s *Impl) MkDir(ctx context.Context, path string) error {
	if err := s.provider.MkDir(ctx, path); err != nil {
		return fmt.Errorf("storage.MkDir(%s): %w", path, err)
	}

	return nil
}

func (s *Impl) Compose(ctx context.Context, path string, chunks []string) error {
	if err := s.provider.Compose(ctx, path, chunks); err != nil {
		return fmt.Errorf("storage.Compose(%s, %s): %w", path, chunks, err)
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
	url, err := s.urlProvider.GetPublicUrl(ctx, path)
	if err != nil {
		return "", fmt.Errorf("storage.GetPublicUrl(%s): %w", path, err)
	}

	return url, nil
}
