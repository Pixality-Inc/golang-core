package gcs

import (
	"context"
	"io"

	"github.com/pixality-inc/golang-core/storage"
	"github.com/pixality-inc/golang-core/util"
)

type StorageProvider struct {
	gcs Client
}

func NewStorageProvider(gcs Client) *StorageProvider {
	return &StorageProvider{
		gcs: gcs,
	}
}

func (p *StorageProvider) FileExists(ctx context.Context, path string) (bool, error) {
	_, result, err := p.gcs.FileExists(ctx, path)

	return result, err
}

func (p *StorageProvider) DeleteFile(ctx context.Context, path string) error {
	return p.gcs.Delete(ctx, path)
}

func (p *StorageProvider) DeleteDir(ctx context.Context, path string) error {
	return p.gcs.DeleteDir(ctx, path)
}

func (p *StorageProvider) Write(ctx context.Context, path string, file io.Reader) error {
	return p.gcs.Upload(ctx, path, file)
}

func (p *StorageProvider) ReadFile(ctx context.Context, path string) (io.ReadCloser, error) {
	return p.gcs.Download(ctx, path)
}

func (p *StorageProvider) ReadDir(_ context.Context, path string) ([]storage.DirEntry, error) {
	return nil, util.ErrNotImplemented
}

func (p *StorageProvider) MkDir(ctx context.Context, path string) error {
	// @todo
	return nil
}

func (p *StorageProvider) Compose(ctx context.Context, path string, chunks []string) error {
	return p.gcs.Compose(ctx, path, chunks)
}

func (p *StorageProvider) GetPublicUrl(ctx context.Context, path string) (string, error) {
	return p.gcs.GetPublicUrl(ctx, path)
}

//nolint:unparam
func (p *StorageProvider) Close() error {
	p.gcs.Close()

	return nil
}
