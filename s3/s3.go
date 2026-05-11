package s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"sync"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"

	"github.com/pixality-inc/golang-core/logger"
	"github.com/pixality-inc/golang-core/storage"
)

// MinPartSize is the S3 minimum part size for multipart uploads (16 MiB).
// All parts of a multipart upload except the very last one must be >= MinPartSize.
const MinPartSize int64 = 16 * 1024 * 1024

// DefaultUploadPartSize is the default part size used by PutObject for auto-multipart.
const DefaultUploadPartSize int64 = 64 * 1024 * 1024

// DefaultUploadConcurrency is the default number of concurrent parts uploaded.
const DefaultUploadConcurrency = 4

// ErrNoChunks is returned by CompleteMultipartUpload when the caller passes an empty chunks slice.
var ErrNoChunks = errors.New("s3: complete multipart called with no chunks")

// ErrBulkDelete is returned by DeleteDir when the bulk delete reported per-object errors.
var ErrBulkDelete = errors.New("s3: bulk delete reported per-object errors")

// ErrEmptyDeletePrefix is returned by DeleteDir when both baseDir and the
// caller-supplied objectName are empty. Proceeding would list every key in
// the bucket and delete all of them — almost always a misconfiguration.
var ErrEmptyDeletePrefix = errors.New("s3: refusing DeleteDir with empty prefix (would wipe the whole bucket)")

// ErrEmptyEndpoint is returned by init when the caller passed no endpoint.
// minio-go would otherwise default to AWS, which is virtually never the
// intent for the S3-compatible providers this package targets.
var ErrEmptyEndpoint = errors.New("s3: empty endpoint")

type Client interface {
	Close()

	Upload(ctx context.Context, objectName string, file io.Reader) error
	UploadFile(ctx context.Context, objectName string, filename string) error

	DeleteDir(ctx context.Context, objectName string) error
	Delete(ctx context.Context, objectName string) error

	Download(ctx context.Context, objectName string) (io.ReadCloser, error)
	DownloadFile(ctx context.Context, objectName string, filename string) error

	FileExists(ctx context.Context, objectName string) (bool, error)

	ReadDir(ctx context.Context, objectName string) ([]storage.DirEntry, error)

	CreateMultipartUpload(ctx context.Context, objectName string) (storage.MultipartUpload, error)
	UploadMultipartChunk(ctx context.Context, objectName string, upload storage.MultipartUpload, chunkNumber int, body io.Reader, size int64) (storage.MultipartChunk, error)
	CompleteMultipartUpload(ctx context.Context, objectName string, upload storage.MultipartUpload, chunks []storage.MultipartChunk) error
	AbortMultipartUpload(ctx context.Context, objectName string, upload storage.MultipartUpload) error

	GetPublicUrl(ctx context.Context, objectName string) (string, error)
}

type Impl struct {
	log           logger.Loggable
	name          string
	bucketName    string
	baseDir       string
	basePublicUrl string

	endpoint     string
	region       string
	accessKey    string
	secretKey    string
	usePathStyle bool

	client *minio.Client
	mutex  sync.Mutex
}

func NewClient(
	name string,
	endpoint string,
	region string,
	accessKey string,
	secretKey string,
	bucketName string,
	baseDir string,
	basePublicUrl string,
	usePathStyle bool,
) Client {
	return &Impl{
		log: logger.NewLoggableImplWithServiceAndFields(
			"s3",
			logger.Fields{
				"name":   name,
				"bucket": bucketName,
			},
		),
		name:          name,
		bucketName:    bucketName,
		baseDir:       baseDir,
		basePublicUrl: basePublicUrl,
		endpoint:      endpoint,
		region:        region,
		accessKey:     accessKey,
		secretKey:     secretKey,
		usePathStyle:  usePathStyle,
	}
}

func (c *Impl) Close() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	c.client = nil
}

func (c *Impl) Upload(ctx context.Context, objectName string, file io.Reader) error {
	log := c.log.GetLogger(ctx)

	log.Infof("Uploading object '%s'", objectName)

	if err := c.init(ctx); err != nil {
		return err
	}

	objectFullName := c.getObjectFullName(objectName)

	metadata, err := storage.GetFileMetadataByName(objectFullName)
	if err != nil {
		return fmt.Errorf("failed to get metadata for %q: %w", objectFullName, err)
	}

	opts := minio.PutObjectOptions{
		ContentType:     metadata.ContentType(),
		ContentEncoding: metadata.ContentEncoding(),
		PartSize:        uint64(DefaultUploadPartSize),
		NumThreads:      uint(DefaultUploadConcurrency),
	}

	if _, err := c.client.PutObject(ctx, c.bucketName, objectFullName, file, -1, opts); err != nil {
		return fmt.Errorf("s3: upload '%s': %w", objectFullName, err)
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

func (c *Impl) Delete(ctx context.Context, objectName string) error {
	c.log.GetLogger(ctx).Infof("Deleting object '%s'", objectName)

	if err := c.init(ctx); err != nil {
		return err
	}

	objectFullName := c.getObjectFullName(objectName)

	if err := c.client.RemoveObject(ctx, c.bucketName, objectFullName, minio.RemoveObjectOptions{}); err != nil {
		return fmt.Errorf("s3: delete '%s': %w", objectFullName, err)
	}

	return nil
}

func (c *Impl) DeleteDir(ctx context.Context, objectName string) error {
	log := c.log.GetLogger(ctx)

	log.Infof("Deleting directory '%s'", objectName)

	if c.baseDir == "" && objectName == "" {
		return ErrEmptyDeletePrefix
	}

	if err := c.init(ctx); err != nil {
		return err
	}

	objectFullName := c.getObjectFullName(objectName)

	listCh := c.client.ListObjects(ctx, c.bucketName, minio.ListObjectsOptions{
		Prefix:    objectFullName,
		Recursive: true,
	})

	// Filter list errors out of the stream so RemoveObjects only sees keys.
	toDelete := make(chan minio.ObjectInfo)
	listErrCh := make(chan error, 1)

	go func() {
		defer close(toDelete)

		for info := range listCh {
			if info.Err != nil {
				listErrCh <- info.Err

				return
			}

			toDelete <- info
		}

		listErrCh <- nil
	}()

	removeCh := c.client.RemoveObjects(ctx, c.bucketName, toDelete, minio.RemoveObjectsOptions{})

	var (
		firstRemoveErr *minio.RemoveObjectError
		removeErrCount int
	)

	for re := range removeCh {
		if firstRemoveErr == nil {
			captured := re
			firstRemoveErr = &captured
		}

		removeErrCount++
	}

	if listErr := <-listErrCh; listErr != nil {
		return fmt.Errorf("s3: list '%s': %w", objectFullName, listErr)
	}

	if firstRemoveErr != nil {
		return fmt.Errorf(
			"%w: under '%s': %d errors, first key=%q: %w",
			ErrBulkDelete,
			objectFullName,
			removeErrCount,
			firstRemoveErr.ObjectName,
			firstRemoveErr.Err,
		)
	}

	return nil
}

func (c *Impl) Download(ctx context.Context, objectName string) (io.ReadCloser, error) {
	log := c.log.GetLogger(ctx)

	log.Infof("Downloading object '%s'", objectName)

	if err := c.init(ctx); err != nil {
		return nil, err
	}

	objectFullName := c.getObjectFullName(objectName)

	obj, err := c.client.GetObject(ctx, c.bucketName, objectFullName, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("s3: download '%s': %w", objectFullName, err)
	}

	// minio-go's GetObject is lazy — the request fires on first Read/Stat.
	// Force the round-trip now so callers see "not found" / auth errors here
	// instead of in the middle of streaming the body.
	if _, err := obj.Stat(); err != nil {
		_ = obj.Close()

		return nil, fmt.Errorf("s3: download '%s': %w", objectFullName, err)
	}

	return obj, nil
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

func (c *Impl) FileExists(ctx context.Context, objectName string) (bool, error) {
	if err := c.init(ctx); err != nil {
		return false, err
	}

	objectFullName := c.getObjectFullName(objectName)

	_, err := c.client.StatObject(ctx, c.bucketName, objectFullName, minio.StatObjectOptions{})
	if err != nil {
		if isNotFoundErr(err) {
			return false, nil
		}

		return false, fmt.Errorf("s3: head '%s': %w", objectFullName, err)
	}

	return true, nil
}

// isNotFoundErr reports whether err represents a 404 for an S3 object.
// minio-go normalizes errors from AWS S3 / MinIO / Hetzner / etc. into
// minio.ErrorResponse via ToErrorResponse — both NoSuchKey and a raw
// HTTP 404 status are accepted to cover all S3-compatible providers.
func isNotFoundErr(err error) bool {
	if err == nil {
		return false
	}

	resp := minio.ToErrorResponse(err)

	return resp.Code == "NoSuchKey" || resp.StatusCode == http.StatusNotFound
}

func (c *Impl) GetPublicUrl(_ context.Context, objectName string) (string, error) {
	objectFullName := c.getObjectFullName(objectName)

	return fmt.Sprintf("%s/%s", c.basePublicUrl, objectFullName), nil
}

func (c *Impl) CreateMultipartUpload(ctx context.Context, objectName string) (storage.MultipartUpload, error) {
	log := c.log.GetLogger(ctx)

	log.Infof("Creating multipart upload for '%s'", objectName)

	if err := c.init(ctx); err != nil {
		return nil, err
	}

	targetFullName := c.getObjectFullName(objectName)

	core := minio.Core{Client: c.client}

	uploadID, err := core.NewMultipartUpload(ctx, c.bucketName, targetFullName, minio.PutObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("s3: create multipart for '%s': %w", targetFullName, err)
	}

	return storage.NewMultipartUpload(uploadID), nil
}

func (c *Impl) UploadMultipartChunk(ctx context.Context, objectName string, upload storage.MultipartUpload, chunkNumber int, body io.Reader, size int64) (storage.MultipartChunk, error) {
	if err := c.init(ctx); err != nil {
		return nil, err
	}

	targetFullName := c.getObjectFullName(objectName)

	core := minio.Core{Client: c.client}

	part, err := core.PutObjectPart(
		ctx,
		c.bucketName,
		targetFullName,
		upload.Id(),
		chunkNumber,
		body,
		size,
		minio.PutObjectPartOptions{},
	)
	if err != nil {
		return nil, fmt.Errorf("s3: upload-part %d for '%s': %w", chunkNumber, targetFullName, err)
	}

	return storage.NewMultipartChunk(chunkNumber, part.ETag), nil
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

	targetFullName := c.getObjectFullName(objectName)

	parts := make([]minio.CompletePart, 0, len(chunks))
	for _, chunk := range chunks {
		parts = append(parts, minio.CompletePart{
			PartNumber: chunk.Number(),
			ETag:       chunk.ETag(),
		})
	}

	core := minio.Core{Client: c.client}

	_, err := core.CompleteMultipartUpload(ctx, c.bucketName, targetFullName, upload.Id(), parts, minio.PutObjectOptions{})
	if err != nil {
		return fmt.Errorf("s3: complete multipart for '%s': %w", targetFullName, err)
	}

	return nil
}

func (c *Impl) AbortMultipartUpload(ctx context.Context, objectName string, upload storage.MultipartUpload) error {
	if err := c.init(ctx); err != nil {
		return err
	}

	targetFullName := c.getObjectFullName(objectName)

	core := minio.Core{Client: c.client}

	if err := core.AbortMultipartUpload(ctx, c.bucketName, targetFullName, upload.Id()); err != nil {
		return fmt.Errorf("s3: abort multipart for '%s': %w", targetFullName, err)
	}

	return nil
}

func (c *Impl) init(ctx context.Context) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.client != nil {
		return nil
	}

	host, secure, err := parseEndpoint(c.endpoint)
	if err != nil {
		return err
	}

	opts := &minio.Options{
		Creds:  credentials.NewStaticV4(c.accessKey, c.secretKey, ""),
		Secure: secure,
		Region: c.region,
	}

	if c.usePathStyle {
		opts.BucketLookup = minio.BucketLookupPath
	}

	client, err := minio.New(host, opts)
	if err != nil {
		return fmt.Errorf("s3: init minio client: %w", err)
	}

	c.client = client

	_ = ctx

	return nil
}

// parseEndpoint splits a caller-supplied endpoint into a host (no scheme) plus
// a Secure flag, since minio.New takes them separately. An empty endpoint is
// rejected — minio-go would default to AWS, which is almost never what the
// caller wants for an S3-compatible setup.
func parseEndpoint(endpoint string) (string, bool, error) {
	if endpoint == "" {
		return "", false, ErrEmptyEndpoint
	}

	parsed, err := url.Parse(endpoint)
	if err != nil {
		return "", false, fmt.Errorf("s3: parse endpoint %q: %w", endpoint, err)
	}

	if parsed.Host == "" {
		// no scheme present, treat the whole string as host; default to https
		return endpoint, true, nil
	}

	return parsed.Host, parsed.Scheme != "http", nil
}

func (c *Impl) getObjectFullName(objectName string) string {
	if c.baseDir != "" {
		return c.baseDir + "/" + objectName
	}

	return objectName
}
