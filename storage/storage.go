package storage

import (
	"context"
	"io"
	"io/fs"
)

type DirEntry interface {
	IsDir() bool
	Name() string
	Type() fs.FileMode
	Info() (fs.FileInfo, error)
}

// MultipartChunk identifies a chunk that has been uploaded as part of a
// multipart upload. ETag is opaque to callers; providers that need a
// content tag (S3) populate it, providers that don't (local/gcs) may
// leave it empty.
type MultipartChunk struct {
	Number int
	ETag   string
}

type Provider interface {
	FileExists(ctx context.Context, path string) (bool, error)
	DeleteFile(ctx context.Context, path string) error
	DeleteDir(ctx context.Context, path string) error
	Write(ctx context.Context, path string, file io.Reader) error
	ReadFile(ctx context.Context, path string) (io.ReadCloser, error)
	ReadDir(ctx context.Context, path string) ([]DirEntry, error)
	MkDir(ctx context.Context, path string) error

	CreateMultipartUpload(ctx context.Context, path string) (uploadId string, err error)
	UploadMultipartChunk(ctx context.Context, path, uploadId string, chunkNumber int, body io.Reader, size int64) (etag string, err error)
	CompleteMultipartUpload(ctx context.Context, path, uploadId string, chunks []MultipartChunk) error
	AbortMultipartUpload(ctx context.Context, path, uploadId string) error

	Close() error
}

type UrlProvider interface {
	GetPublicUrl(ctx context.Context, path string) (string, error)
}

//go:generate mockgen -destination mocks/storage_gen.go -source storage.go
type Storage interface {
	FileExists(ctx context.Context, path string) (bool, error)
	DeleteFile(ctx context.Context, path string) error
	DeleteDir(ctx context.Context, path string) error
	Write(ctx context.Context, path string, file io.Reader) error
	WriteFile(ctx context.Context, path string, filename string) error
	ReadFile(ctx context.Context, path string) (io.ReadCloser, error)
	DownloadFile(ctx context.Context, path string, filename string) error
	ReadDir(ctx context.Context, path string) ([]DirEntry, error)
	MkDir(ctx context.Context, path string) error

	CreateMultipartUpload(ctx context.Context, path string) (uploadId string, err error)
	UploadMultipartChunk(ctx context.Context, path, uploadId string, chunkNumber int, body io.Reader, size int64) (etag string, err error)
	CompleteMultipartUpload(ctx context.Context, path, uploadId string, chunks []MultipartChunk) error
	AbortMultipartUpload(ctx context.Context, path, uploadId string) error

	GetPublicUrl(ctx context.Context, path string) (string, error)
	Close() error
}
