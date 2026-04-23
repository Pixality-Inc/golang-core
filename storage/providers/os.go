package providers

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pixality-inc/golang-core/logger"
	"github.com/pixality-inc/golang-core/storage"
)

var (
	ErrNoChunksProvided = errors.New("no chunks provided")
	ErrChunkProcess     = errors.New("chunk process")
)

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

func (p *OsProvider) Compose(ctx context.Context, path string, chunks []string) error {
	log := logger.GetLogger(ctx)

	if len(chunks) == 0 {
		return ErrNoChunksProvided
	}

	destPath := p.getFullPath(path)

	destDir := filepath.Dir(destPath)

	if err := os.MkdirAll(destDir, os.ModePerm); err != nil {
		return fmt.Errorf("create dir %s for file %s: %w", destDir, path, err)
	}

	if len(chunks) == 1 {
		return os.Rename(p.getFullPath(chunks[0]), destPath)
	}

	tmpFile, err := os.CreateTemp(destDir, ".compose-*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}

	tmpName := tmpFile.Name()

	defer func() {
		if fErr := tmpFile.Close(); fErr != nil {
			log.WithError(fErr).Errorf("failed to close temp file '%s'", tmpName)
		}

		if rmErr := os.Remove(tmpName); rmErr != nil && !errors.Is(rmErr, os.ErrNotExist) {
			log.WithError(rmErr).Errorf("failed to remove temp file '%s'", tmpName)
		}
	}()

	copyChunkToTemp := func(chunkPath string) error {
		sourcePath := p.getFullPath(chunkPath)

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

	for _, chunkPath := range chunks {
		if err = copyChunkToTemp(chunkPath); err != nil {
			return fmt.Errorf("%w: %s: %w", ErrChunkProcess, chunkPath, err)
		}
	}

	if err = os.Rename(tmpName, destPath); err != nil {
		return fmt.Errorf("rename temp file to destination: %w", err)
	}

	return nil
}

func (p *OsProvider) Close() error {
	return nil
}

func (p *OsProvider) LocalPath(_ context.Context, path string) (string, error) {
	return p.getFullPath(path), nil
}

func (p *OsProvider) getFullPath(path string) string {
	return filepath.Join(p.dir, path)
}
