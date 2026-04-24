package gcs

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"

	gcs "cloud.google.com/go/storage"
	"google.golang.org/api/iterator"

	"github.com/pixality-inc/golang-core/storage"
)

// ReadDir lists immediate children under objectName. GCS has no real
// directories, so the result is built from the Objects listing with
// Delimiter="/": ObjectAttrs with a non-empty Prefix become dir entries,
// the rest become file entries. Names are returned relative to objectName
// (tail only) and sorted by name, matching os.ReadDir semantics.
func (c *Impl) ReadDir(ctx context.Context, objectName string) ([]storage.DirEntry, error) {
	log := c.log.GetLogger(ctx)

	log.Infof("Listing directory '%s'", objectName)

	if err := c.init(ctx); err != nil {
		return nil, err
	}

	prefix := listPrefix(c.getObjectFullName(objectName))

	objectIter := c.client.Bucket(c.bucketName).Objects(ctx, &gcs.Query{
		Prefix:    prefix,
		Delimiter: "/",
	})

	var entries []storage.DirEntry

	for {
		attrs, err := objectIter.Next()
		if errors.Is(err, iterator.Done) {
			break
		}

		if err != nil {
			return nil, fmt.Errorf("gcs: list '%s': %w", prefix, err)
		}

		if entry := dirEntryFromAttrs(attrs, prefix); entry != nil {
			entries = append(entries, entry)
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	return entries, nil
}

// dirEntryFromAttrs maps one ObjectAttrs from a Delimiter="/" listing into a
// storage entry relative to prefix. Returns nil for entries that should be
// skipped: the prefix itself, empty names, and zero-byte "directory marker"
// objects whose key ends with "/".
func dirEntryFromAttrs(attrs *gcs.ObjectAttrs, prefix string) storage.DirEntry {
	if attrs == nil {
		return nil
	}

	if attrs.Prefix != "" {
		name := strings.TrimSuffix(strings.TrimPrefix(attrs.Prefix, prefix), "/")
		if name == "" {
			return nil
		}

		return storage.NewDirEntry(name)
	}

	name := strings.TrimPrefix(attrs.Name, prefix)
	if name == "" {
		// the prefix itself materialized as a zero-byte object, skip
		return nil
	}

	if strings.HasSuffix(attrs.Name, "/") {
		// zero-byte "directory marker" object; a sibling CommonPrefix will
		// already surface this as a dir entry
		return nil
	}

	return storage.NewFileEntry(name, attrs.Size, attrs.Updated)
}

// listPrefix normalizes a resolved full object name into a GCS listing prefix
// with a trailing slash, so Delimiter="/" groups immediate children. An empty
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
