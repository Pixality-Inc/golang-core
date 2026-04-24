package s3

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"

	"github.com/pixality-inc/golang-core/storage"
)

// ReadDir lists immediate children under objectName. S3 has no real directories,
// so the result is built from ListObjectsV2 with Delimiter="/": Contents become
// file entries, CommonPrefixes become dir entries. Names are returned relative
// to objectName (tail only) and sorted by name, matching os.ReadDir semantics.
func (c *Impl) ReadDir(ctx context.Context, objectName string) ([]storage.DirEntry, error) {
	log := c.log.GetLogger(ctx)

	log.Infof("Listing directory '%s'", objectName)

	if err := c.init(ctx); err != nil {
		return nil, err
	}

	prefix := listPrefix(c.getObjectFullName(objectName))

	paginator := awss3.NewListObjectsV2Paginator(c.client, &awss3.ListObjectsV2Input{
		Bucket:    aws.String(c.bucketName),
		Prefix:    aws.String(prefix),
		Delimiter: aws.String("/"),
	})

	var entries []storage.DirEntry

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("s3: list '%s': %w", prefix, err)
		}

		entries = append(entries, dirEntriesFromPage(page.CommonPrefixes, page.Contents, prefix)...)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	return entries, nil
}

// dirEntriesFromPage converts a single ListObjectsV2 page into storage entries
// relative to prefix. CommonPrefixes become dir entries; Contents become file
// entries. Zero-byte "directory marker" keys ending with "/" are skipped so
// they do not duplicate the corresponding CommonPrefixes dir entry.
func dirEntriesFromPage(commonPrefixes []types.CommonPrefix, contents []types.Object, prefix string) []storage.DirEntry {
	entries := make([]storage.DirEntry, 0, len(commonPrefixes)+len(contents))

	for _, cp := range commonPrefixes {
		name := strings.TrimSuffix(strings.TrimPrefix(aws.ToString(cp.Prefix), prefix), "/")
		if name == "" {
			continue
		}

		entries = append(entries, &dirEntry{
			name:  name,
			isDir: true,
		})
	}

	for _, obj := range contents {
		key := aws.ToString(obj.Key)

		name := strings.TrimPrefix(key, prefix)

		if name == "" {
			// the prefix itself materialized as a zero-byte object, skip
			continue
		}

		if strings.HasSuffix(key, "/") {
			// zero-byte "directory marker" object; CommonPrefixes already
			// covers this as a dir entry
			continue
		}

		entries = append(entries, &dirEntry{
			name:    name,
			size:    aws.ToInt64(obj.Size),
			modTime: aws.ToTime(obj.LastModified),
		})
	}

	return entries
}

// listPrefix normalizes a resolved full object name into a ListObjectsV2 prefix
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
