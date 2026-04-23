package providers

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/pixality-inc/golang-core/storage"
)

var ErrStorageFailed = errors.New("storage failed")

type SyncImpl struct {
	storages []storage.Storage
}

func NewSync(storages ...storage.Storage) *SyncImpl {
	return &SyncImpl{
		storages: storages,
	}
}

func (s *SyncImpl) FileExists(ctx context.Context, path string) (bool, error) {
	existsEverywhere := true

	for _, entry := range s.storages {
		result, err := entry.FileExists(ctx, path)
		if err == nil {
			return result, nil
		}

		existsEverywhere = existsEverywhere && result
	}

	return existsEverywhere, nil
}

func (s *SyncImpl) DeleteFile(ctx context.Context, path string) error {
	for _, entry := range s.storages {
		if err := entry.DeleteFile(ctx, path); err != nil {
			return fmt.Errorf("%w: failed to delete file %s: %w", ErrStorageFailed, path, err)
		}
	}

	return nil
}

func (s *SyncImpl) DeleteDir(ctx context.Context, path string) error {
	for _, entry := range s.storages {
		if err := entry.DeleteDir(ctx, path); err != nil {
			return fmt.Errorf("%w: failed to delete directory %s: %w", ErrStorageFailed, path, err)
		}
	}

	return nil
}

func (s *SyncImpl) Write(ctx context.Context, path string, file io.Reader) error {
	for _, entry := range s.storages {
		if err := entry.Write(ctx, path, file); err != nil {
			return fmt.Errorf("%w: failed to write file %s: %w", ErrStorageFailed, path, err)
		}
	}

	return nil
}

func (s *SyncImpl) ReadFile(ctx context.Context, path string) (io.ReadCloser, error) {
	for _, entry := range s.storages {
		if file, err := entry.ReadFile(ctx, path); err == nil {
			return file, nil
		}
	}

	return nil, fmt.Errorf("%w: failed to read file %s", ErrStorageFailed, path)
}

func (s *SyncImpl) ReadDir(ctx context.Context, path string) ([]storage.DirEntry, error) {
	for _, entry := range s.storages {
		if entries, err := entry.ReadDir(ctx, path); err != nil {
			return nil, fmt.Errorf("%w: failed to read directory %s: %w", ErrStorageFailed, path, err)
		} else {
			return entries, nil
		}
	}

	return nil, fmt.Errorf("%w: failed to read directory %s", ErrStorageFailed, path)
}

func (s *SyncImpl) MkDir(ctx context.Context, path string) error {
	for _, entry := range s.storages {
		if err := entry.MkDir(ctx, path); err != nil {
			return fmt.Errorf("%w: failed to create directory %s: %w", ErrStorageFailed, path, err)
		}
	}

	return nil
}

func (s *SyncImpl) Compose(ctx context.Context, path string, chunks []string) error {
	for _, entry := range s.storages {
		if err := entry.Compose(ctx, path, chunks); err != nil {
			return fmt.Errorf("%w: failed to compose file %s from %d chunks: %w", ErrStorageFailed, path, len(chunks), err)
		}
	}

	return nil
}

func (s *SyncImpl) Close() error {
	for _, entry := range s.storages {
		if err := entry.Close(); err != nil {
			return fmt.Errorf("%w: failed to close storage: %w", ErrStorageFailed, err)
		}
	}

	return nil
}
