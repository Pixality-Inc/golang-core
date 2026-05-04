package sftp

import (
	"context"
	"io"

	"github.com/pixality-inc/golang-core/storage"
)

type StorageProvider struct {
	sftp Client
}

func NewStorageProvider(sftp Client) *StorageProvider {
	return &StorageProvider{
		sftp: sftp,
	}
}

func (p *StorageProvider) FileExists(ctx context.Context, path string) (bool, error) {
	return p.sftp.FileExists(ctx, path)
}

func (p *StorageProvider) DeleteFile(ctx context.Context, path string) error {
	return p.sftp.Delete(ctx, path)
}

func (p *StorageProvider) DeleteDir(ctx context.Context, path string) error {
	return p.sftp.DeleteDir(ctx, path)
}

func (p *StorageProvider) Write(ctx context.Context, path string, file io.Reader) error {
	return p.sftp.Upload(ctx, path, file)
}

func (p *StorageProvider) ReadFile(ctx context.Context, path string) (io.ReadCloser, error) {
	return p.sftp.Download(ctx, path)
}

func (p *StorageProvider) ReadDir(ctx context.Context, path string) ([]storage.DirEntry, error) {
	return p.sftp.ReadDir(ctx, path)
}

func (p *StorageProvider) MkDir(ctx context.Context, path string) error {
	return p.sftp.MkDir(ctx, path)
}

func (p *StorageProvider) Compose(ctx context.Context, path string, chunks []string) error {
	return p.sftp.Compose(ctx, path, chunks)
}

func (p *StorageProvider) GetPublicUrl(ctx context.Context, path string) (string, error) {
	return p.sftp.GetPublicUrl(ctx, path)
}

//nolint:unparam
func (p *StorageProvider) Close() error {
	p.sftp.Close()

	return nil
}
