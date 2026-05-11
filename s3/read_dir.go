package s3

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/minio/minio-go/v7"

	"github.com/pixality-inc/golang-core/storage"
)

// ReadDir lists immediate children under objectName. S3 has no real directories,
// so the result is built from ListObjects with Delimiter implicit via Recursive=false:
// keys ending in "/" are CommonPrefixes (dirs), the rest are files. Names are returned
// relative to objectName (tail only) and sorted by name, matching os.ReadDir semantics.
func (c *Impl) ReadDir(ctx context.Context, objectName string) ([]storage.DirEntry, error) {
	log := c.log.GetLogger(ctx)

	log.Infof("Listing directory '%s'", objectName)

	if err := c.init(ctx); err != nil {
		return nil, err
	}

	prefix := listPrefix(c.getObjectFullName(objectName))

	listCh := c.client.ListObjects(ctx, c.bucketName, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: false,
	})

	infos := make([]minio.ObjectInfo, 0)

	for info := range listCh {
		if info.Err != nil {
			return nil, fmt.Errorf("s3: list '%s': %w", prefix, info.Err)
		}

		infos = append(infos, info)
	}

	entries := dirEntriesFromObjects(infos, prefix)

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	return entries, nil
}

// dirEntriesFromObjects converts a flat slice of ObjectInfo (as yielded by
// ListObjects with Recursive=false) into storage entries relative to prefix.
// Keys ending in "/" become dir entries; other keys become file entries.
// The zero-byte object representing the prefix itself is skipped, and dir
// entries are deduplicated by name in case a provider returns both a
// CommonPrefix and a zero-byte directory marker for the same path.
func dirEntriesFromObjects(infos []minio.ObjectInfo, prefix string) []storage.DirEntry {
	entries := make([]storage.DirEntry, 0, len(infos))
	seenDirs := make(map[string]struct{}, len(infos))

	for _, info := range infos {
		key := info.Key

		if key == prefix {
			// the prefix itself materialized as a zero-byte object
			continue
		}

		if strings.HasSuffix(key, "/") {
			name := strings.TrimSuffix(strings.TrimPrefix(key, prefix), "/")
			if name == "" {
				continue
			}

			if _, dup := seenDirs[name]; dup {
				continue
			}

			seenDirs[name] = struct{}{}
			entries = append(entries, storage.NewDirEntry(name))

			continue
		}

		name := strings.TrimPrefix(key, prefix)
		if name == "" {
			continue
		}

		entries = append(entries, storage.NewFileEntry(name, info.Size, info.LastModified))
	}

	return entries
}

// listPrefix normalizes a resolved full object name into a ListObjects prefix
// with a trailing slash, so listings group immediate children. An empty
// full name lists the whole bucket root.
func listPrefix(fullName string) string {
	if fullName == "" {
		return ""
	}

	if strings.HasSuffix(fullName, "/") {
		return fullName
	}

	return fullName + "/"
}
