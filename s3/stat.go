package s3

import (
	"context"
	"fmt"
	"io/fs"
	"path"

	"github.com/minio/minio-go/v7"

	"github.com/pixality-inc/golang-core/storage"
)

// Stat returns metadata for one exact object using a HEAD request. Virtual
// directories inferred by ReadDir are not objects and cannot be statted.
func (c *Impl) Stat(ctx context.Context, objectName string) (fs.FileInfo, error) {
	if err := c.init(ctx); err != nil {
		return nil, err
	}

	objectFullName := c.getObjectFullName(objectName)

	info, err := c.client.StatObject(ctx, c.bucketName, objectFullName, minio.StatObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("s3: stat '%s': %w", objectFullName, err)
	}

	return storage.NewFileEntry(path.Base(objectFullName), info.Size, info.LastModified).Info()
}
