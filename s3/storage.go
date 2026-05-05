package s3

import (
	"context"
	"io"

	"github.com/pixality-inc/golang-core/storage"
)

type StorageProvider struct {
	s3 Client
}

func NewStorageProvider(s3 Client) *StorageProvider {
	return &StorageProvider{
		s3: s3,
	}
}

func (p *StorageProvider) FileExists(ctx context.Context, path string) (bool, error) {
	return p.s3.FileExists(ctx, path)
}

func (p *StorageProvider) DeleteFile(ctx context.Context, path string) error {
	return p.s3.Delete(ctx, path)
}

func (p *StorageProvider) DeleteDir(ctx context.Context, path string) error {
	return p.s3.DeleteDir(ctx, path)
}

func (p *StorageProvider) Write(ctx context.Context, path string, file io.Reader) error {
	return p.s3.Upload(ctx, path, file)
}

func (p *StorageProvider) ReadFile(ctx context.Context, path string) (io.ReadCloser, error) {
	return p.s3.Download(ctx, path)
}

func (p *StorageProvider) ReadDir(ctx context.Context, path string) ([]storage.DirEntry, error) {
	return p.s3.ReadDir(ctx, path)
}

func (p *StorageProvider) MkDir(_ context.Context, _ string) error {
	// s3 has no directories, paths are flat keys
	return nil
}

func (p *StorageProvider) CreateMultipartUpload(ctx context.Context, path string) (string, error) {
	return p.s3.CreateMultipartUpload(ctx, path)
}

func (p *StorageProvider) UploadMultipartChunk(ctx context.Context, path, uploadId string, chunkNumber int, body io.Reader, size int64) (string, error) {
	return p.s3.UploadMultipartChunk(ctx, path, uploadId, chunkNumber, body, size)
}

func (p *StorageProvider) CompleteMultipartUpload(ctx context.Context, path, uploadId string, chunks []storage.MultipartChunk) error {
	return p.s3.CompleteMultipartUpload(ctx, path, uploadId, chunks)
}

func (p *StorageProvider) AbortMultipartUpload(ctx context.Context, path, uploadId string) error {
	return p.s3.AbortMultipartUpload(ctx, path, uploadId)
}

func (p *StorageProvider) GetPublicUrl(ctx context.Context, path string) (string, error) {
	return p.s3.GetPublicUrl(ctx, path)
}

//nolint:unparam
func (p *StorageProvider) Close() error {
	p.s3.Close()

	return nil
}
