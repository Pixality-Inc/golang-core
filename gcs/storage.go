package gcs

import (
	"context"
	"io"

	"github.com/pixality-inc/golang-core/storage"
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

func (p *StorageProvider) ReadDir(ctx context.Context, path string) ([]storage.DirEntry, error) {
	return p.gcs.ReadDir(ctx, path)
}

func (p *StorageProvider) MkDir(ctx context.Context, path string) error {
	// @todo
	return nil
}

func (p *StorageProvider) CreateMultipartUpload(ctx context.Context, path string) (string, error) {
	return p.gcs.CreateMultipartUpload(ctx, path)
}

func (p *StorageProvider) UploadMultipartChunk(ctx context.Context, path, uploadId string, chunkNumber int, body io.Reader, size int64) (string, error) {
	return p.gcs.UploadMultipartChunk(ctx, path, uploadId, chunkNumber, body, size)
}

func (p *StorageProvider) CompleteMultipartUpload(ctx context.Context, path, uploadId string, chunks []storage.MultipartChunk) error {
	return p.gcs.CompleteMultipartUpload(ctx, path, uploadId, chunks)
}

func (p *StorageProvider) AbortMultipartUpload(ctx context.Context, path, uploadId string) error {
	return p.gcs.AbortMultipartUpload(ctx, path, uploadId)
}

func (p *StorageProvider) GetPublicUrl(ctx context.Context, path string) (string, error) {
	return p.gcs.GetPublicUrl(ctx, path)
}

//nolint:unparam
func (p *StorageProvider) Close() error {
	p.gcs.Close()

	return nil
}
