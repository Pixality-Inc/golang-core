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

// MultipartUpload identifies an in-progress multipart upload returned by
// CreateMultipartUpload. Providers may attach additional state by
// implementing this interface with their own type.
type MultipartUpload interface {
	Id() string
}

// MultipartChunk identifies a chunk that has been uploaded as part of a
// multipart upload. ETag is opaque to callers; providers that need a
// content tag (S3) populate it, providers that don't (local/gcs) may
// leave it empty.
type MultipartChunk interface {
	Number() int
	ETag() string
}

type multipartUpload struct {
	id string
}

func (u multipartUpload) Id() string { return u.id }

func NewMultipartUpload(id string) MultipartUpload {
	return multipartUpload{id: id}
}

type multipartChunk struct {
	number int
	etag   string
}

func (c multipartChunk) Number() int  { return c.number }
func (c multipartChunk) ETag() string { return c.etag }

func NewMultipartChunk(number int, etag string) MultipartChunk {
	return multipartChunk{number: number, etag: etag}
}

type Provider interface {
	FileExists(ctx context.Context, path string) (bool, error)
	DeleteFile(ctx context.Context, path string) error
	DeleteDir(ctx context.Context, path string) error
	Write(ctx context.Context, path string, file io.Reader) error
	ReadFile(ctx context.Context, path string) (io.ReadCloser, error)
	ReadDir(ctx context.Context, path string) ([]DirEntry, error)
	MkDir(ctx context.Context, path string) error

	CreateMultipartUpload(ctx context.Context, path string) (MultipartUpload, error)
	UploadMultipartChunk(ctx context.Context, path string, upload MultipartUpload, chunkNumber int, body io.Reader, size int64) (MultipartChunk, error)
	CompleteMultipartUpload(ctx context.Context, path string, upload MultipartUpload, chunks []MultipartChunk) error
	AbortMultipartUpload(ctx context.Context, path string, upload MultipartUpload) error

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

	CreateMultipartUpload(ctx context.Context, path string) (MultipartUpload, error)
	UploadMultipartChunk(ctx context.Context, path string, upload MultipartUpload, chunkNumber int, body io.Reader, size int64) (MultipartChunk, error)
	CompleteMultipartUpload(ctx context.Context, path string, upload MultipartUpload, chunks []MultipartChunk) error
	AbortMultipartUpload(ctx context.Context, path string, upload MultipartUpload) error

	GetPublicUrl(ctx context.Context, path string) (string, error)
	Close() error
}
