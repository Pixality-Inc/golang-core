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

// syncMultipartUploadId is the opaque uploadId returned by SyncImpl. It
// carries the per-storage uploadIds in order so that subsequent calls
// can dispatch to each child storage without keeping local state.
type syncMultipartUploadId struct {
	Ids []string `json:"ids"`
}

func (s *SyncImpl) CreateMultipartUpload(ctx context.Context, path string) (string, error) {
	if len(s.storages) == 0 {
		return "", fmt.Errorf("%w: no storages configured", ErrStorageFailed)
	}

	ids := make([]string, 0, len(s.storages))

	for i, entry := range s.storages {
		id, err := entry.CreateMultipartUpload(ctx, path)
		if err != nil {
			for j := range i {
				if abortErr := s.storages[j].AbortMultipartUpload(ctx, path, ids[j]); abortErr != nil {
					logger.GetLogger(ctx).WithError(abortErr).Errorf("sync storage rollback: failed to abort %s", path)
				}
			}

			return "", fmt.Errorf("%w: failed to create multipart for %s: %w", ErrStorageFailed, path, err)
		}

		ids = append(ids, id)
	}

	return encodeSyncUploadId(ids)
}

func (s *SyncImpl) UploadMultipartChunk(ctx context.Context, path, uploadId string, chunkNumber int, body io.Reader, size int64) (string, error) {
	if len(s.storages) == 0 {
		return "", fmt.Errorf("%w: no storages configured", ErrStorageFailed)
	}

	ids, err := decodeSyncUploadId(uploadId)
	if err != nil {
		return "", fmt.Errorf("%w: %w", ErrStorageFailed, err)
	}

	if len(ids) != len(s.storages) {
		return "", fmt.Errorf("%w: uploadId carries %d ids, %d storages configured", ErrStorageFailed, len(ids), len(s.storages))
	}

	log := logger.GetLogger(ctx)

	tmpFile, err := os.CreateTemp("", "sync-upload-part-*")
	if err != nil {
		return "", fmt.Errorf("%w: failed to create temp file for %s: %w", ErrStorageFailed, path, err)
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

		return "", fmt.Errorf("%w: failed to spool source for %s: %w", ErrStorageFailed, path, err)
	}

	if err = tmpFile.Sync(); err != nil {
		if closeErr := tmpFile.Close(); closeErr != nil {
			log.WithError(closeErr).Errorf("sync storage: failed to close temp spool file %s", tmpName)
		}

		return "", fmt.Errorf("%w: failed to sync spool for %s: %w", ErrStorageFailed, path, err)
	}

	if err = tmpFile.Close(); err != nil {
		return "", fmt.Errorf("%w: failed to close spool for %s: %w", ErrStorageFailed, path, err)
	}

	etags := make([]string, 0, len(s.storages))

	for i, entry := range s.storages {
		spoolReader, openErr := os.Open(tmpName)
		if openErr != nil {
			return "", fmt.Errorf("%w: failed to open spool for %s: %w", ErrStorageFailed, path, openErr)
		}

		etag, partErr := entry.UploadMultipartChunk(ctx, path, ids[i], chunkNumber, spoolReader, size)
		closeErr := spoolReader.Close()

		if partErr != nil {
			return "", fmt.Errorf("%w: failed to upload chunk %d for %s: %w", ErrStorageFailed, chunkNumber, path, partErr)
		}

		if closeErr != nil {
			return "", fmt.Errorf("%w: failed to close spool reader for %s: %w", ErrStorageFailed, path, closeErr)
		}

		etags = append(etags, etag)
	}

	return encodeSyncEtags(etags)
}

func (s *SyncImpl) CompleteMultipartUpload(ctx context.Context, path, uploadId string, chunks []storage.MultipartChunk) error {
	ids, err := decodeSyncUploadId(uploadId)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrStorageFailed, err)
	}

	if len(ids) != len(s.storages) {
		return fmt.Errorf("%w: uploadId carries %d ids, %d storages configured", ErrStorageFailed, len(ids), len(s.storages))
	}

	for i, entry := range s.storages {
		childChunks := make([]storage.MultipartChunk, 0, len(chunks))

		for _, chunk := range chunks {
			etags, decErr := decodeSyncEtags(chunk.ETag)
			if decErr != nil {
				return fmt.Errorf("%w: chunk %d etag: %w", ErrStorageFailed, chunk.Number, decErr)
			}

			if len(etags) != len(s.storages) {
				return fmt.Errorf("%w: chunk %d carries %d etags, %d storages configured", ErrStorageFailed, chunk.Number, len(etags), len(s.storages))
			}

			childChunks = append(childChunks, storage.MultipartChunk{
				Number: chunk.Number,
				ETag:   etags[i],
			})
		}

		if err = entry.CompleteMultipartUpload(ctx, path, ids[i], childChunks); err != nil {
			return fmt.Errorf("%w: failed to complete multipart for %s: %w", ErrStorageFailed, path, err)
		}
	}

	return nil
}

func (s *SyncImpl) AbortMultipartUpload(ctx context.Context, path, uploadId string) error {
	ids, err := decodeSyncUploadId(uploadId)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrStorageFailed, err)
	}

	if len(ids) != len(s.storages) {
		return fmt.Errorf("%w: uploadId carries %d ids, %d storages configured", ErrStorageFailed, len(ids), len(s.storages))
	}

	var errs []error

	for i, entry := range s.storages {
		if abortErr := entry.AbortMultipartUpload(ctx, path, ids[i]); abortErr != nil {
			errs = append(errs, abortErr)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("%w: abort multipart: %w", ErrStorageFailed, errors.Join(errs...))
	}

	return nil
}

func encodeSyncUploadId(ids []string) (string, error) {
	encoded, err := json.Marshal(syncMultipartUploadId{Ids: ids})
	if err != nil {
		return "", fmt.Errorf("encode sync uploadId: %w", err)
	}

	return string(encoded), nil
}

func decodeSyncUploadId(s string) ([]string, error) {
	var v syncMultipartUploadId
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return nil, fmt.Errorf("decode sync uploadId: %w", err)
	}

	return v.Ids, nil
}

func encodeSyncEtags(etags []string) (string, error) {
	encoded, err := json.Marshal(etags)
	if err != nil {
		return "", fmt.Errorf("encode sync etags: %w", err)
	}

	return string(encoded), nil
}

func decodeSyncEtags(s string) ([]string, error) {
	var v []string
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return nil, fmt.Errorf("decode sync etags: %w", err)
	}

	return v, nil
}

func (s *SyncImpl) Close() error {
	for _, entry := range s.storages {
		if err := entry.Close(); err != nil {
			return fmt.Errorf("%w: failed to close storage: %w", ErrStorageFailed, err)
		}
	}

	return nil
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
