package storage

import (
	"context"
	"fmt"

	"github.com/pixality-inc/golang-core/logger"
)

//go:generate mockgen -destination mocks/local_storage_gen.go -source local_storage.go
type LocalStorage interface {
	Storage

	LocalPath(ctx context.Context, path string) (string, error)
}

type LocalStorageProvider interface {
	Provider

	LocalPath(ctx context.Context, path string) (string, error)
}

type LocalStorageImpl struct {
	Storage

	log         logger.Loggable
	provider    LocalStorageProvider
	urlProvider UrlProvider
}

func NewLocalStorage(provider LocalStorageProvider, urlProvider UrlProvider) LocalStorage {
	return &LocalStorageImpl{
		Storage:     NewStorage(provider, urlProvider),
		log:         logger.NewLoggableImplWithService("local_storage"),
		provider:    provider,
		urlProvider: urlProvider,
	}
}

func (s *LocalStorageImpl) LocalPath(ctx context.Context, path string) (string, error) {
	result, err := s.provider.LocalPath(ctx, path)
	if err != nil {
		return "", fmt.Errorf("local_storage.LocalPath(%s): %w", path, err)
	}

	return result, nil
}
