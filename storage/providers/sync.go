package providers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"

	"github.com/pixality-inc/golang-core/logger"
	"github.com/pixality-inc/golang-core/storage"
)

var ErrStorageFailed = errors.New("storage failed")

type SyncImpl struct {
	storages []storage.Storage
}

func NewSync(storages ...storage.Storage) *SyncImpl {
	return &SyncImpl{
		storages: storages,
	}
}

func (s *SyncImpl) FileExists(ctx context.Context, path string) (bool, error) {
	if len(s.storages) == 0 {
		return false, fmt.Errorf("%w: no storages configured", ErrStorageFailed)
	}

	for _, entry := range s.storages {
		ok, err := entry.FileExists(ctx, path)
		if err != nil {
			return false, fmt.Errorf("%w: failed to check file %s: %w", ErrStorageFailed, path, err)
		}

		if !ok {
			return false, nil
		}
	}

	return true, nil
}

func (s *SyncImpl) DeleteFile(ctx context.Context, path string) error {
	for _, entry := range s.storages {
		if err := entry.DeleteFile(ctx, path); err != nil {
			return fmt.Errorf("%w: failed to delete file %s: %w", ErrStorageFailed, path, err)
		}
	}

	return nil
}

func (s *SyncImpl) DeleteDir(ctx context.Context, path string) error {
	for _, entry := range s.storages {
		if err := entry.DeleteDir(ctx, path); err != nil {
			return fmt.Errorf("%w: failed to delete directory %s: %w", ErrStorageFailed, path, err)
		}
	}

	return nil
}

func (s *SyncImpl) Write(ctx context.Context, path string, file io.Reader) error {
	if len(s.storages) == 0 {
		return fmt.Errorf("%w: no storages configured", ErrStorageFailed)
	}

	log := logger.GetLogger(ctx)

	tmpFile, err := os.CreateTemp("", "sync-write-*")
	if err != nil {
		return fmt.Errorf("%w: failed to create temp file for %s: %w", ErrStorageFailed, path, err)
	}

	tmpName := tmpFile.Name()

	defer func() {
		if rmErr := os.Remove(tmpName); rmErr != nil && !errors.Is(rmErr, os.ErrNotExist) {
			log.WithError(rmErr).Errorf("sync storage: failed to remove temp spool file %s", tmpName)
		}
	}()

	if _, err = io.Copy(tmpFile, file); err != nil {
		if closeErr := tmpFile.Close(); closeErr != nil {
			log.WithError(closeErr).Errorf("sync storage: failed to close temp spool file %s", tmpName)
		}

		return fmt.Errorf("%w: failed to spool source for %s: %w", ErrStorageFailed, path, err)
	}

	if err = tmpFile.Sync(); err != nil {
		if closeErr := tmpFile.Close(); closeErr != nil {
			log.WithError(closeErr).Errorf("sync storage: failed to close temp spool file %s", tmpName)
		}

		return fmt.Errorf("%w: failed to sync spool for %s: %w", ErrStorageFailed, path, err)
	}

	if err = tmpFile.Close(); err != nil {
		return fmt.Errorf("%w: failed to close spool for %s: %w", ErrStorageFailed, path, err)
	}

	var written []storage.Storage

	for _, entry := range s.storages {
		spoolReader, openErr := os.Open(tmpName)
		if openErr != nil {
			return fmt.Errorf("%w: failed to open spool for %s: %w", ErrStorageFailed, path, openErr)
		}

		writeErr := entry.Write(ctx, path, spoolReader)
		closeErr := spoolReader.Close()

		if writeErr != nil {
			for _, w := range written {
				if delErr := w.DeleteFile(ctx, path); delErr != nil {
					log.WithError(delErr).Errorf("sync storage rollback: failed to delete %s", path)
				}
			}

			return fmt.Errorf("%w: failed to write file %s: %w", ErrStorageFailed, path, writeErr)
		}

		if closeErr != nil {
			for _, w := range written {
				if delErr := w.DeleteFile(ctx, path); delErr != nil {
					log.WithError(delErr).Errorf("sync storage rollback: failed to delete %s", path)
				}
			}

			if delErr := entry.DeleteFile(ctx, path); delErr != nil {
				log.WithError(delErr).Errorf("sync storage rollback: failed to delete %s", path)
			}

			return fmt.Errorf("%w: failed to close spool reader for %s: %w", ErrStorageFailed, path, closeErr)
		}

		written = append(written, entry)
	}

	return nil
}

func (s *SyncImpl) ReadFile(ctx context.Context, path string) (io.ReadCloser, error) {
	if len(s.storages) == 0 {
		return nil, fmt.Errorf("%w: no storages configured", ErrStorageFailed)
	}

	var errs []error

	for _, entry := range s.storages {
		file, err := entry.ReadFile(ctx, path)
		if err == nil {
			return file, nil
		}

		errs = append(errs, err)
	}

	return nil, fmt.Errorf("%w: failed to read file %s: %w", ErrStorageFailed, path, errors.Join(errs...))
}

func (s *SyncImpl) ReadDir(ctx context.Context, path string) ([]storage.DirEntry, error) {
	if len(s.storages) == 0 {
		return nil, fmt.Errorf("%w: failed to read directory %s", ErrStorageFailed, path)
	}

	var first []storage.DirEntry

	for i, entry := range s.storages {
		entries, err := entry.ReadDir(ctx, path)
		if err != nil {
			return nil, fmt.Errorf("%w: failed to read directory %s: %w", ErrStorageFailed, path, err)
		}

		if i == 0 {
			first = entries

			continue
		}

		if !dirEntriesMatch(first, entries) {
			return nil, fmt.Errorf("%w: directory %s listing mismatch between storages", ErrStorageFailed, path)
		}
	}

	return first, nil
}

func (s *SyncImpl) MkDir(ctx context.Context, path string) error {
	for _, entry := range s.storages {
		if err := entry.MkDir(ctx, path); err != nil {
			return fmt.Errorf("%w: failed to create directory %s: %w", ErrStorageFailed, path, err)
		}
	}

	return nil
}

// syncMultipartUpload carries the per-storage uploads in order so that
// subsequent calls can dispatch to each child storage without keeping
// local state. Its Id() is a JSON-encoded copy of the per-storage ids,
// kept for backward-compatible logging/diagnostics.
type syncMultipartUpload struct {
	uploads []storage.MultipartUpload
}

func (u *syncMultipartUpload) Id() string {
	ids := make([]string, len(u.uploads))
	for i, up := range u.uploads {
		ids[i] = up.Id()
	}

	encoded, err := json.Marshal(ids)
	if err != nil {
		return ""
	}

	return string(encoded)
}

// syncMultipartChunk fans an uploaded chunk out across child storages.
type syncMultipartChunk struct {
	number int
	chunks []storage.MultipartChunk
}

func (c *syncMultipartChunk) Number() int { return c.number }

func (c *syncMultipartChunk) ETag() string {
	etags := make([]string, len(c.chunks))
	for i, ch := range c.chunks {
		etags[i] = ch.ETag()
	}

	encoded, err := json.Marshal(etags)
	if err != nil {
		return ""
	}

	return string(encoded)
}

func (s *SyncImpl) CreateMultipartUpload(ctx context.Context, path string) (storage.MultipartUpload, error) {
	if len(s.storages) == 0 {
		return nil, fmt.Errorf("%w: no storages configured", ErrStorageFailed)
	}

	uploads := make([]storage.MultipartUpload, 0, len(s.storages))

	for i, entry := range s.storages {
		upload, err := entry.CreateMultipartUpload(ctx, path)
		if err != nil {
			for j := range i {
				if abortErr := s.storages[j].AbortMultipartUpload(ctx, path, uploads[j]); abortErr != nil {
					logger.GetLogger(ctx).WithError(abortErr).Errorf("sync storage rollback: failed to abort %s", path)
				}
			}

			return nil, fmt.Errorf("%w: failed to create multipart for %s: %w", ErrStorageFailed, path, err)
		}

		uploads = append(uploads, upload)
	}

	return &syncMultipartUpload{uploads: uploads}, nil
}

func (s *SyncImpl) UploadMultipartChunk(ctx context.Context, path string, upload storage.MultipartUpload, chunkNumber int, body io.Reader, size int64) (storage.MultipartChunk, error) {
	if len(s.storages) == 0 {
		return nil, fmt.Errorf("%w: no storages configured", ErrStorageFailed)
	}

	syncUpload, err := s.castUpload(upload)
	if err != nil {
		return nil, err
	}

	log := logger.GetLogger(ctx)

	tmpFile, err := os.CreateTemp("", "sync-upload-part-*")
	if err != nil {
		return nil, fmt.Errorf("%w: failed to create temp file for %s: %w", ErrStorageFailed, path, err)
	}

	tmpName := tmpFile.Name()

	defer func() {
		if rmErr := os.Remove(tmpName); rmErr != nil && !errors.Is(rmErr, os.ErrNotExist) {
			log.WithError(rmErr).Errorf("sync storage: failed to remove temp spool file %s", tmpName)
		}
	}()

	if _, err = io.Copy(tmpFile, body); err != nil {
		if closeErr := tmpFile.Close(); closeErr != nil {
			log.WithError(closeErr).Errorf("sync storage: failed to close temp spool file %s", tmpName)
		}

		return nil, fmt.Errorf("%w: failed to spool source for %s: %w", ErrStorageFailed, path, err)
	}

	if err = tmpFile.Sync(); err != nil {
		if closeErr := tmpFile.Close(); closeErr != nil {
			log.WithError(closeErr).Errorf("sync storage: failed to close temp spool file %s", tmpName)
		}

		return nil, fmt.Errorf("%w: failed to sync spool for %s: %w", ErrStorageFailed, path, err)
	}

	if err = tmpFile.Close(); err != nil {
		return nil, fmt.Errorf("%w: failed to close spool for %s: %w", ErrStorageFailed, path, err)
	}

	chunks := make([]storage.MultipartChunk, 0, len(s.storages))

	for i, entry := range s.storages {
		spoolReader, openErr := os.Open(tmpName)
		if openErr != nil {
			return nil, fmt.Errorf("%w: failed to open spool for %s: %w", ErrStorageFailed, path, openErr)
		}

		chunk, partErr := entry.UploadMultipartChunk(ctx, path, syncUpload.uploads[i], chunkNumber, spoolReader, size)
		closeErr := spoolReader.Close()

		if partErr != nil {
			return nil, fmt.Errorf("%w: failed to upload chunk %d for %s: %w", ErrStorageFailed, chunkNumber, path, partErr)
		}

		if closeErr != nil {
			return nil, fmt.Errorf("%w: failed to close spool reader for %s: %w", ErrStorageFailed, path, closeErr)
		}

		chunks = append(chunks, chunk)
	}

	return &syncMultipartChunk{number: chunkNumber, chunks: chunks}, nil
}

func (s *SyncImpl) CompleteMultipartUpload(ctx context.Context, path string, upload storage.MultipartUpload, chunks []storage.MultipartChunk) error {
	syncUpload, err := s.castUpload(upload)
	if err != nil {
		return err
	}

	for i, entry := range s.storages {
		childChunks := make([]storage.MultipartChunk, 0, len(chunks))

		for _, chunk := range chunks {
			syncChunk, ok := chunk.(*syncMultipartChunk)
			if !ok {
				return fmt.Errorf("%w: chunk %d is not a sync chunk", ErrStorageFailed, chunk.Number())
			}

			if len(syncChunk.chunks) != len(s.storages) {
				return fmt.Errorf("%w: chunk %d carries %d entries, %d storages configured", ErrStorageFailed, chunk.Number(), len(syncChunk.chunks), len(s.storages))
			}

			childChunks = append(childChunks, syncChunk.chunks[i])
		}

		if err = entry.CompleteMultipartUpload(ctx, path, syncUpload.uploads[i], childChunks); err != nil {
			return fmt.Errorf("%w: failed to complete multipart for %s: %w", ErrStorageFailed, path, err)
		}
	}

	return nil
}

func (s *SyncImpl) AbortMultipartUpload(ctx context.Context, path string, upload storage.MultipartUpload) error {
	syncUpload, err := s.castUpload(upload)
	if err != nil {
		return err
	}

	var errs []error

	for i, entry := range s.storages {
		if abortErr := entry.AbortMultipartUpload(ctx, path, syncUpload.uploads[i]); abortErr != nil {
			errs = append(errs, abortErr)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("%w: abort multipart: %w", ErrStorageFailed, errors.Join(errs...))
	}

	return nil
}

func (s *SyncImpl) Close() error {
	for _, entry := range s.storages {
		if err := entry.Close(); err != nil {
			return fmt.Errorf("%w: failed to close storage: %w", ErrStorageFailed, err)
		}
	}

	return nil
}

func (s *SyncImpl) castUpload(upload storage.MultipartUpload) (*syncMultipartUpload, error) {
	syncUpload, ok := upload.(*syncMultipartUpload)
	if !ok {
		return nil, fmt.Errorf("%w: upload is not a sync upload", ErrStorageFailed)
	}

	if len(syncUpload.uploads) != len(s.storages) {
		return nil, fmt.Errorf("%w: upload carries %d entries, %d storages configured", ErrStorageFailed, len(syncUpload.uploads), len(s.storages))
	}

	return syncUpload, nil
}

func dirEntriesMatch(aEntries []storage.DirEntry, bEntries []storage.DirEntry) bool {
	if len(aEntries) != len(bEntries) {
		return false
	}

	sigsA := make([]string, len(aEntries))
	for i, e := range aEntries {
		sigsA[i] = fmt.Sprintf("%s|%t|%d", e.Name(), e.IsDir(), e.Type())
	}

	sigsB := make([]string, len(bEntries))
	for i, e := range bEntries {
		sigsB[i] = fmt.Sprintf("%s|%t|%d", e.Name(), e.IsDir(), e.Type())
	}

	slices.Sort(sigsA)
	slices.Sort(sigsB)

	return slices.Equal(sigsA, sigsB)
}
