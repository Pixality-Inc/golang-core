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

func (p *StorageProvider) CreateMultipartUpload(ctx context.Context, path string) (storage.MultipartUpload, error) {
	return p.gcs.CreateMultipartUpload(ctx, path)
}

func (p *StorageProvider) UploadMultipartChunk(ctx context.Context, path string, upload storage.MultipartUpload, chunkNumber int, body io.Reader, size int64) (storage.MultipartChunk, error) {
	return p.gcs.UploadMultipartChunk(ctx, path, upload, chunkNumber, body, size)
}

func (p *StorageProvider) CompleteMultipartUpload(ctx context.Context, path string, upload storage.MultipartUpload, chunks []storage.MultipartChunk) error {
	return p.gcs.CompleteMultipartUpload(ctx, path, upload, chunks)
}

func (p *StorageProvider) AbortMultipartUpload(ctx context.Context, path string, upload storage.MultipartUpload) error {
	return p.gcs.AbortMultipartUpload(ctx, path, upload)
}

func (p *StorageProvider) GetPublicUrl(ctx context.Context, path string) (string, error) {
	return p.gcs.GetPublicUrl(ctx, path)
}

//nolint:unparam
func (p *StorageProvider) Close() error {
	p.gcs.Close()

	return nil
}
