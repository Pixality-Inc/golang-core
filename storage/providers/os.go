package providers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/pixality-inc/golang-core/logger"
	"github.com/pixality-inc/golang-core/storage"
)

var (
	ErrNoChunksProvided = errors.New("no chunks provided")
	ErrChunkProcess     = errors.New("chunk process")
)

// multipartPartsSuffix is appended to the target path to form a sibling
// directory that holds in-progress multipart chunks. The directory is
// removed on Complete or Abort.
const multipartPartsSuffix = ".parts"

type OsProvider struct {
	dir string
}

func NewOsProvider(dir string) storage.LocalStorageProvider {
	return &OsProvider{
		dir: dir,
	}
}

func (p *OsProvider) FileExists(ctx context.Context, path string) (bool, error) {
	fullPath := p.getFullPath(path)

	_, err := os.Stat(fullPath)
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	} else if err != nil {
		return false, err
	}

	return true, nil
}

func (p *OsProvider) DeleteFile(ctx context.Context, path string) error {
	fullPath := p.getFullPath(path)

	if err := os.Remove(fullPath); err != nil {
		return err
	}

	return nil
}

func (p *OsProvider) DeleteDir(ctx context.Context, path string) error {
	fullPath := p.getFullPath(path)

	if err := os.RemoveAll(fullPath); err != nil {
		return err
	}

	return nil
}

func (p *OsProvider) Write(ctx context.Context, path string, file io.Reader) error {
	fullPath := p.getFullPath(path)

	dirName := filepath.Dir(fullPath)
	if err := os.MkdirAll(dirName, os.ModePerm); err != nil {
		return fmt.Errorf("create dir %s for file %s: %w", dirName, path, err)
	}

	destFile, err := os.Create(fullPath)
	if err != nil {
		return fmt.Errorf("create file %s: %w", path, err)
	}

	defer func() {
		if fErr := destFile.Close(); fErr != nil {
			logger.GetLogger(ctx).WithError(fErr).Errorf("failed to close file '%s'", fullPath)
		}
	}()

	if _, err = io.Copy(destFile, file); err != nil {
		return fmt.Errorf("copy file to '%s': %w", fullPath, err)
	}

	return nil
}

func (p *OsProvider) ReadFile(_ context.Context, path string) (io.ReadCloser, error) {
	fullPath := p.getFullPath(path)

	file, err := os.Open(fullPath)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}

	return file, nil
}

func (p *OsProvider) ReadDir(_ context.Context, path string) ([]storage.DirEntry, error) {
	fullPath := p.getFullPath(path)

	dirEntries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, fmt.Errorf("read dir: %w", err)
	}

	results := make([]storage.DirEntry, len(dirEntries))

	for index, dirEntry := range dirEntries {
		results[index] = dirEntry
	}

	return results, nil
}

func (p *OsProvider) MkDir(_ context.Context, path string) error {
	fullPath := p.getFullPath(path)

	//nolint:gosec
	if err := os.MkdirAll(fullPath, os.ModePerm); err != nil {
		return fmt.Errorf("mkdir: %w", err)
	}

	return nil
}

func (p *OsProvider) CreateMultipartUpload(_ context.Context, _ string) (storage.MultipartUpload, error) {
	return storage.NewMultipartUpload(uuid.New().String()), nil
}

func (p *OsProvider) UploadMultipartChunk(ctx context.Context, path string, upload storage.MultipartUpload, chunkNumber int, body io.Reader, _ int64) (storage.MultipartChunk, error) {
	chunkPath := p.multipartChunkPath(path, upload.Id(), chunkNumber)

	if err := p.Write(ctx, chunkPath, body); err != nil {
		return nil, fmt.Errorf("write chunk %d: %w", chunkNumber, err)
	}

	return storage.NewMultipartChunk(chunkNumber, chunkPath), nil
}

func (p *OsProvider) CompleteMultipartUpload(ctx context.Context, path string, upload storage.MultipartUpload, chunks []storage.MultipartChunk) error {
	log := logger.GetLogger(ctx)

	if len(chunks) == 0 {
		return ErrNoChunksProvided
	}

	destPath := p.getFullPath(path)

	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, os.ModePerm); err != nil {
		return fmt.Errorf("create dir %s for file %s: %w", destDir, path, err)
	}

	tmpFile, err := os.CreateTemp(destDir, ".compose-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}

	tmpName := tmpFile.Name()

	defer func() {
		if rmErr := os.Remove(tmpName); rmErr != nil && !errors.Is(rmErr, os.ErrNotExist) {
			log.WithError(rmErr).Errorf("failed to remove temp file '%s'", tmpName)
		}
	}()

	copyChunkToTemp := func(chunkNumber int) error {
		sourcePath := p.getFullPath(p.multipartChunkPath(path, upload.Id(), chunkNumber))

		chunkFile, err := os.Open(sourcePath)
		if err != nil {
			return fmt.Errorf("open chunk: %w", err)
		}

		defer func() {
			if fErr := chunkFile.Close(); fErr != nil {
				log.WithError(fErr).Errorf("failed to close chunk source: %s", sourcePath)
			}
		}()

		if _, err = io.Copy(tmpFile, chunkFile); err != nil {
			return fmt.Errorf("copy chunk: %w", err)
		}

		return nil
	}

	for _, chunk := range chunks {
		if err = copyChunkToTemp(chunk.Number()); err != nil {
			return fmt.Errorf("%w: chunk %d: %w", ErrChunkProcess, chunk.Number(), err)
		}
	}

	if err = tmpFile.Sync(); err != nil {
		return fmt.Errorf("sync temp file: %w", err)
	}

	if err = tmpFile.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	if err = os.Rename(tmpName, destPath); err != nil {
		return fmt.Errorf("rename temp file to destination: %w", err)
	}

	if rmErr := p.cleanupMultipartDir(ctx, path, upload.Id()); rmErr != nil && !errors.Is(rmErr, os.ErrNotExist) {
		log.WithError(rmErr).Errorf("failed to clean parts dir for '%s/%s'", path, upload.Id())
	}

	return nil
}

func (p *OsProvider) AbortMultipartUpload(ctx context.Context, path string, upload storage.MultipartUpload) error {
	if err := p.cleanupMultipartDir(ctx, path, upload.Id()); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("abort multipart: %w", err)
	}

	return nil
}

func (p *OsProvider) Close() error {
	return nil
}

func (p *OsProvider) LocalPath(_ context.Context, path string) (string, error) {
	return p.getFullPath(path), nil
}

// cleanupMultipartDir removes the per-upload parts directory and, if the
// target's parts parent directory is now empty, removes that as well.
// The parent removal is best-effort: a non-empty parent just stays.
func (p *OsProvider) cleanupMultipartDir(ctx context.Context, path, uploadId string) error {
	if err := p.DeleteDir(ctx, p.multipartUploadDir(path, uploadId)); err != nil {
		return err
	}

	parentRel := path + multipartPartsSuffix
	parentFull := p.getFullPath(parentRel)

	if err := os.Remove(parentFull); err != nil && !errors.Is(err, os.ErrNotExist) {
		logger.GetLogger(ctx).WithError(err).Debugf("multipart parent dir '%s' not removed", parentRel)
	}

	return nil
}

func (p *OsProvider) multipartUploadDir(path, uploadId string) string {
	return path + multipartPartsSuffix + "/" + uploadId
}

func (p *OsProvider) multipartChunkPath(path, uploadId string, chunkNumber int) string {
	return fmt.Sprintf("%s/%d", p.multipartUploadDir(path, uploadId), chunkNumber)
}

func (p *OsProvider) getFullPath(path string) string {
	return filepath.Join(p.dir, path)
}
