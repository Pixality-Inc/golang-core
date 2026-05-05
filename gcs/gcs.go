package gcs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/google/uuid"
	"github.com/pixality-inc/golang-core/logger"
	storage "github.com/pixality-inc/golang-core/storage"

	gcs "cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

// gcsMaxComposeSources is the GCS Compose API limit on the number of source
// objects per call. Larger uploads are composed in two passes via group
// objects.
const gcsMaxComposeSources = 32

// multipartPartsSuffix is appended to the target objectName to form the
// prefix that holds in-progress multipart chunks. The prefix is removed on
// CompleteMultipartUpload or AbortMultipartUpload.
const multipartPartsSuffix = ".parts"

// ErrNoChunks is returned by CompleteMultipartUpload when the caller passes an empty chunks slice.
var ErrNoChunks = errors.New("gcs: complete multipart called with no chunks")

type Client interface {
	Close()

	Upload(ctx context.Context, objectName string, file io.Reader) error

	UploadFile(ctx context.Context, objectName string, filename string) error

	DeleteDir(ctx context.Context, objectName string) error

	Delete(ctx context.Context, objectName string) error

	Download(ctx context.Context, objectName string) (io.ReadCloser, error)

	DownloadFile(ctx context.Context, objectName string, filename string) error

	FileExists(ctx context.Context, objectName string) (*gcs.ObjectAttrs, bool, error)

	ReadDir(ctx context.Context, objectName string) ([]storage.DirEntry, error)

	CreateMultipartUpload(ctx context.Context, objectName string) (storage.MultipartUpload, error)
	UploadMultipartChunk(ctx context.Context, objectName string, upload storage.MultipartUpload, chunkNumber int, body io.Reader, size int64) (storage.MultipartChunk, error)
	CompleteMultipartUpload(ctx context.Context, objectName string, upload storage.MultipartUpload, chunks []storage.MultipartChunk) error
	AbortMultipartUpload(ctx context.Context, objectName string, upload storage.MultipartUpload) error

	GetPublicUrl(ctx context.Context, objectName string) (string, error)
}

type Impl struct {
	log                 logger.Loggable
	credentialsFilename string
	name                string
	bucketName          string
	baseDir             string
	basePublicUrl       string
	client              *gcs.Client
	mutex               sync.Mutex
}

func NewClient(
	credentialsFilename string,
	name string,
	bucketName string,
	baseDir string,
	basePublicUrl string,
) Client {
	return &Impl{
		log: logger.NewLoggableImplWithServiceAndFields(
			"gcs",
			logger.Fields{
				"name":   name,
				"bucket": bucketName,
			},
		),
		credentialsFilename: credentialsFilename,
		name:                name,
		bucketName:          bucketName,
		baseDir:             baseDir,
		basePublicUrl:       basePublicUrl,
		client:              nil,
		mutex:               sync.Mutex{},
	}
}

func (c *Impl) Close() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.client == nil {
		return
	}

	if err := c.client.Close(); err != nil {
		c.log.GetLoggerWithoutContext().WithError(err).Error("close failed")
	}
}

func (c *Impl) Upload(ctx context.Context, objectName string, file io.Reader) error {
	log := c.log.GetLogger(ctx)

	log.Infof("Uploading object '%s'", objectName)

	if err := c.init(ctx); err != nil {
		return err
	}

	objectFullName := c.getObjectFullName(objectName)

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	writer := c.client.Bucket(c.bucketName).Object(objectFullName).NewWriter(ctx)

	metadata, err := storage.GetFileMetadataByName(objectFullName)
	if err != nil {
		return fmt.Errorf("failed to get metadata for %q: %w", objectFullName, err)
	}

	contentType := metadata.ContentType()
	contentEncoding := metadata.ContentEncoding()

	if contentType != "" {
		writer.ContentType = contentType
	}

	if contentEncoding != "" {
		writer.ContentEncoding = contentEncoding
	}

	if _, err := io.Copy(writer, file); err != nil {
		cancel()

		if closeErr := writer.Close(); closeErr != nil {
			log.WithError(closeErr).Errorf("failed to close writer for '%s'", objectFullName)
		}

		return err
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close writer for '%s': %w", objectFullName, err)
	}

	return nil
}

func (c *Impl) UploadFile(ctx context.Context, objectName string, filename string) error {
	if err := c.init(ctx); err != nil {
		return err
	}

	file, err := os.Open(filename)
	if err != nil {
		return err
	}

	defer func() {
		if err = file.Close(); err != nil {
			c.log.GetLogger(ctx).WithError(err).Errorf("failed to close file '%s'", filename)
		}
	}()

	return c.Upload(ctx, objectName, file)
}

func (c *Impl) DeleteDir(ctx context.Context, objectName string) error {
	c.log.GetLogger(ctx).Infof("Deleting directory '%s'", objectName)

	if err := c.init(ctx); err != nil {
		return err
	}

	objectFullName := c.getObjectFullName(objectName)

	bucket := c.client.Bucket(c.bucketName)

	it := bucket.Objects(ctx, &gcs.Query{Prefix: objectFullName})

	for {
		attrs, err := it.Next()

		if errors.Is(err, iterator.Done) {
			break
		}

		if err != nil {
			return err
		}

		if err := bucket.Object(attrs.Name).Delete(ctx); err != nil {
			return err
		}
	}

	return nil
}

func (c *Impl) Delete(ctx context.Context, objectName string) error {
	c.log.GetLogger(ctx).Infof("Deleting object '%s'", objectName)

	if err := c.init(ctx); err != nil {
		return err
	}

	objectFullName := c.getObjectFullName(objectName)

	object := c.client.Bucket(c.bucketName).Object(objectFullName)

	return object.Delete(ctx)
}

func (c *Impl) Download(ctx context.Context, objectName string) (io.ReadCloser, error) {
	log := c.log.GetLogger(ctx)

	log.Infof("Downloading object '%s'", objectName)

	if err := c.init(ctx); err != nil {
		return nil, err
	}

	objectFullName := c.getObjectFullName(objectName)

	readCloser, err := c.client.Bucket(c.bucketName).Object(objectFullName).NewReader(ctx)
	if err != nil {
		return nil, err
	}

	return readCloser, nil
}

func (c *Impl) DownloadFile(ctx context.Context, objectName string, filename string) error {
	log := c.log.GetLogger(ctx)

	readCloser, err := c.Download(ctx, objectName)
	if err != nil {
		return err
	}

	log.Infof("Downloading object '%s' to '%s'", objectName, filename)

	defer func() {
		if err := readCloser.Close(); err != nil {
			log.WithError(err).Errorf("failed to close reader '%s' for '%s'", objectName, filename)
		}
	}()

	outFile, err := os.Create(filename)
	if err != nil {
		return err
	}

	defer func() {
		if err = outFile.Close(); err != nil {
			log.WithError(err).Errorf("failed to close file '%s'", filename)
		}
	}()

	if _, err := io.Copy(outFile, readCloser); err != nil {
		return fmt.Errorf("failed to copy data: %w", err)
	}

	return nil
}

func (c *Impl) FileExists(ctx context.Context, objectName string) (*gcs.ObjectAttrs, bool, error) {
	if err := c.init(ctx); err != nil {
		return nil, false, err
	}

	objectFullName := c.getObjectFullName(objectName)

	attrs, err := c.client.Bucket(c.bucketName).Object(objectFullName).Attrs(ctx)
	if err != nil && errors.Is(err, gcs.ErrObjectNotExist) {
		return nil, false, nil
	} else if err != nil {
		return nil, false, err
	}

	return attrs, true, nil
}

func (c *Impl) CreateMultipartUpload(_ context.Context, _ string) (storage.MultipartUpload, error) {
	return storage.NewMultipartUpload(uuid.New().String()), nil
}

func (c *Impl) UploadMultipartChunk(ctx context.Context, objectName string, upload storage.MultipartUpload, chunkNumber int, body io.Reader, _ int64) (storage.MultipartChunk, error) {
	chunkPath := c.multipartChunkPath(objectName, upload.Id(), chunkNumber)

	if err := c.Upload(ctx, chunkPath, body); err != nil {
		return nil, fmt.Errorf("gcs: upload chunk %d for '%s': %w", chunkNumber, objectName, err)
	}

	return storage.NewMultipartChunk(chunkNumber, chunkPath), nil
}

func (c *Impl) CompleteMultipartUpload(ctx context.Context, objectName string, upload storage.MultipartUpload, chunks []storage.MultipartChunk) error {
	log := c.log.GetLogger(ctx)
	log.Infof("Completing multipart upload '%s' with %d chunks", objectName, len(chunks))

	if len(chunks) == 0 {
		return fmt.Errorf("%w: '%s'", ErrNoChunks, objectName)
	}

	if err := c.init(ctx); err != nil {
		return err
	}

	chunkPaths := make([]string, 0, len(chunks))
	for _, chunk := range chunks {
		chunkPaths = append(chunkPaths, c.multipartChunkPath(objectName, upload.Id(), chunk.Number()))
	}

	if err := c.composeAll(ctx, objectName, upload.Id(), chunkPaths); err != nil {
		return err
	}

	if err := c.DeleteDir(ctx, c.multipartUploadDir(objectName, upload.Id())); err != nil {
		log.WithError(err).Errorf("gcs: failed to clean parts dir for '%s/%s'", objectName, upload.Id())
	}

	return nil
}

func (c *Impl) AbortMultipartUpload(ctx context.Context, objectName string, upload storage.MultipartUpload) error {
	if err := c.DeleteDir(ctx, c.multipartUploadDir(objectName, upload.Id())); err != nil {
		return fmt.Errorf("gcs: abort multipart for '%s/%s': %w", objectName, upload.Id(), err)
	}

	return nil
}

func (c *Impl) GetPublicUrl(ctx context.Context, objectName string) (string, error) {
	if err := c.init(ctx); err != nil {
		return "", err
	}

	objectFullName := c.getObjectFullName(objectName)

	url := fmt.Sprintf("%s/%s", c.basePublicUrl, objectFullName)

	return url, nil
}

// composeAll handles the GCS 32-sources-per-Compose limit by composing
// chunks into intermediate group objects first, then composing the groups
// into the target. Single-pass when the chunk count fits the limit.
func (c *Impl) composeAll(ctx context.Context, objectName, uploadId string, chunkPaths []string) error {
	if len(chunkPaths) <= gcsMaxComposeSources {
		return c.composeOnce(ctx, objectName, chunkPaths)
	}

	groupPaths := make([]string, 0, (len(chunkPaths)+gcsMaxComposeSources-1)/gcsMaxComposeSources)

	for groupId := 0; groupId*gcsMaxComposeSources < len(chunkPaths); groupId++ {
		startIdx := groupId * gcsMaxComposeSources

		endIdx := min(startIdx+gcsMaxComposeSources, len(chunkPaths))

		groupPath := fmt.Sprintf("%s/group_%d", c.multipartUploadDir(objectName, uploadId), groupId)

		if err := c.composeOnce(ctx, groupPath, chunkPaths[startIdx:endIdx]); err != nil {
			return fmt.Errorf("gcs: compose group %d: %w", groupId, err)
		}

		groupPaths = append(groupPaths, groupPath)
	}

	return c.composeOnce(ctx, objectName, groupPaths)
}

func (c *Impl) composeOnce(ctx context.Context, targetName string, sourceNames []string) error {
	bucket := c.client.Bucket(c.bucketName)

	targetFullName := c.getObjectFullName(targetName)

	sourceObjects := make([]*gcs.ObjectHandle, len(sourceNames))

	for i, name := range sourceNames {
		sourceObjects[i] = bucket.Object(c.getObjectFullName(name))
	}

	if _, err := bucket.Object(targetFullName).ComposerFrom(sourceObjects...).Run(ctx); err != nil {
		return fmt.Errorf("gcs: compose '%s' from %d sources: %w", targetFullName, len(sourceNames), err)
	}

	return nil
}

func (c *Impl) multipartUploadDir(objectName, uploadId string) string {
	return objectName + multipartPartsSuffix + "/" + uploadId
}

func (c *Impl) multipartChunkPath(objectName, uploadId string, chunkNumber int) string {
	return fmt.Sprintf("%s/%d", c.multipartUploadDir(objectName, uploadId), chunkNumber)
}

func (c *Impl) init(ctx context.Context) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.client != nil {
		return nil
	}

	client, err := gcs.NewClient(ctx, option.WithCredentialsFile(c.credentialsFilename))
	if err != nil {
		return err
	}

	c.client = client

	return nil
}

func (c *Impl) getObjectFullName(objectName string) string {
	if c.baseDir != "" {
		return c.baseDir + "/" + objectName
	}

	return objectName
}
