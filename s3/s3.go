package s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	awss3 "github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	"github.com/pixality-inc/golang-core/storage"

	"github.com/pixality-inc/golang-core/logger"
)

// MinPartSize is the S3 minimum part size for multipart uploads (5 MiB).
// All parts of a multipart upload except the very last one must be >= MinPartSize.
const MinPartSize int64 = 5 * 1024 * 1024

// DefaultUploadPartSize is the default part size used by the s3 manager Uploader.
const DefaultUploadPartSize int64 = 16 * 1024 * 1024

// DefaultUploadConcurrency is the default number of concurrent parts uploaded.
const DefaultUploadConcurrency = 4

// abortMultipartTimeout bounds the best-effort cleanup of a failed multipart
// upload. It runs on a fresh context so the abort still fires when the caller's
// ctx was the reason the upload failed (e.g. cancellation). Without this, a
// canceled Compose would leak the multipart upload on the bucket.
const abortMultipartTimeout = 30 * time.Second

// ErrChunkTooSmall is returned by Compose when a non-last chunk is smaller than MinPartSize.
var ErrChunkTooSmall = errors.New("s3: non-last compose chunk smaller than MinPartSize")

// ErrNoChunks is returned by Compose when the caller passes an empty chunks slice.
var ErrNoChunks = errors.New("s3: compose called with no chunks")

// ErrBulkDelete is returned by DeleteDir when the S3 API succeeded at the
// request level but reported per-object errors in the response.
var ErrBulkDelete = errors.New("s3: bulk delete reported per-object errors")

// ErrEmptyDeletePrefix is returned by DeleteDir when both baseDir and the
// caller-supplied objectName are empty. Proceeding would list every key in
// the bucket and delete all of them — almost always a misconfiguration.
var ErrEmptyDeletePrefix = errors.New("s3: refusing DeleteDir with empty prefix (would wipe the whole bucket)")

type Client interface {
	Close()

	Upload(ctx context.Context, objectName string, file io.Reader) error
	UploadFile(ctx context.Context, objectName string, filename string) error

	DeleteDir(ctx context.Context, objectName string) error
	Delete(ctx context.Context, objectName string) error

	Download(ctx context.Context, objectName string) (io.ReadCloser, error)
	DownloadFile(ctx context.Context, objectName string, filename string) error

	FileExists(ctx context.Context, objectName string) (bool, error)

	Compose(ctx context.Context, objectName string, chunks []string) error

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

	client *awss3.Client
	//nolint:staticcheck // SA1019: feature/s3/manager mandated by v0.6.14 patch; migrate to transfermanager later.
	uploader *manager.Uploader
	mutex    sync.Mutex
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
	c.uploader = nil
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

	object := &awss3.PutObjectInput{
		Bucket: aws.String(c.bucketName),
		Key:    aws.String(objectFullName),
		Body:   file,
	}

	contentType := metadata.ContentType()
	contentEncoding := metadata.ContentEncoding()

	if contentType != "" {
		object.ContentType = &contentType
	}

	if contentEncoding != "" {
		object.ContentEncoding = &contentEncoding
	}

	//nolint:staticcheck // SA1019: feature/s3/manager mandated by v0.6.14 patch.
	if _, err = c.uploader.Upload(ctx, object); err != nil {
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

	_, err := c.client.DeleteObject(ctx, &awss3.DeleteObjectInput{
		Bucket: aws.String(c.bucketName),
		Key:    aws.String(objectFullName),
	})
	if err != nil {
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

	paginator := awss3.NewListObjectsV2Paginator(c.client, &awss3.ListObjectsV2Input{
		Bucket: aws.String(c.bucketName),
		Prefix: aws.String(objectFullName),
	})

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return fmt.Errorf("s3: list '%s': %w", objectFullName, err)
		}

		if len(page.Contents) == 0 {
			continue
		}

		ids := make([]types.ObjectIdentifier, 0, len(page.Contents))
		for _, obj := range page.Contents {
			ids = append(ids, types.ObjectIdentifier{Key: obj.Key})
		}

		resp, err := c.client.DeleteObjects(ctx, &awss3.DeleteObjectsInput{
			Bucket: aws.String(c.bucketName),
			Delete: &types.Delete{Objects: ids, Quiet: aws.Bool(true)},
		})
		if err != nil {
			return fmt.Errorf("s3: bulk delete under '%s': %w", objectFullName, err)
		}

		if len(resp.Errors) > 0 {
			first := resp.Errors[0]

			return fmt.Errorf(
				"%w: under '%s': %d errors, first key=%q code=%q message=%q",
				ErrBulkDelete,
				objectFullName,
				len(resp.Errors),
				aws.ToString(first.Key),
				aws.ToString(first.Code),
				aws.ToString(first.Message),
			)
		}
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

	out, err := c.client.GetObject(ctx, &awss3.GetObjectInput{
		Bucket: aws.String(c.bucketName),
		Key:    aws.String(objectFullName),
	})
	if err != nil {
		return nil, fmt.Errorf("s3: download '%s': %w", objectFullName, err)
	}

	return out.Body, nil
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

	_, err := c.client.HeadObject(ctx, &awss3.HeadObjectInput{
		Bucket: aws.String(c.bucketName),
		Key:    aws.String(objectFullName),
	})
	if err != nil {
		if isNotFoundErr(err) {
			return false, nil
		}

		return false, fmt.Errorf("s3: head '%s': %w", objectFullName, err)
	}

	return true, nil
}

// isNotFoundErr reports whether err represents a 404 for an S3 object.
// Covers AWS-typed NotFound, smithy APIError with code "NotFound"/"NoSuchKey",
// and HTTP status 404 returned by S3-compatible endpoints (MinIO, Hetzner, etc.).
func isNotFoundErr(err error) bool {
	var nf *types.NotFound
	if errors.As(err, &nf) {
		return true
	}

	var httpErr *smithyhttp.ResponseError
	if errors.As(err, &httpErr) && httpErr.HTTPStatusCode() == http.StatusNotFound {
		return true
	}

	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		case "NotFound", "NoSuchKey":
			return true
		}
	}

	return false
}

func (c *Impl) GetPublicUrl(_ context.Context, objectName string) (string, error) {
	objectFullName := c.getObjectFullName(objectName)

	return fmt.Sprintf("%s/%s", c.basePublicUrl, objectFullName), nil
}

type composeSource struct {
	fullName string
	size     int64
}

func (c *Impl) Compose(ctx context.Context, objectName string, chunks []string) error {
	log := c.log.GetLogger(ctx)

	log.Infof("Composing object '%s' from %d chunks", objectName, len(chunks))

	if len(chunks) == 0 {
		return fmt.Errorf("%w: '%s'", ErrNoChunks, objectName)
	}

	if err := c.init(ctx); err != nil {
		return err
	}

	targetFullName := c.getObjectFullName(objectName)

	sources := make([]composeSource, 0, len(chunks))
	for _, chunk := range chunks {
		full := c.getObjectFullName(chunk)

		head, err := c.client.HeadObject(ctx, &awss3.HeadObjectInput{
			Bucket: aws.String(c.bucketName),
			Key:    aws.String(full),
		})
		if err != nil {
			return fmt.Errorf("s3: head chunk '%s': %w", full, err)
		}

		sources = append(sources, composeSource{
			fullName: full,
			size:     aws.ToInt64(head.ContentLength),
		})
	}

	// Case A: single source -> CopyObject
	if len(sources) == 1 {
		return c.copyObject(ctx, sources[0].fullName, targetFullName)
	}

	// Validate non-last sizes
	for i := range len(sources) - 1 {
		if sources[i].size < MinPartSize {
			return fmt.Errorf(
				"%w: chunk %d ('%s') is %d bytes, target '%s' minimum %d",
				ErrChunkTooSmall, i, sources[i].fullName, sources[i].size, targetFullName, MinPartSize,
			)
		}
	}

	return c.composeMultipart(ctx, targetFullName, sources)
}

func (c *Impl) copyObject(ctx context.Context, sourceFullName, targetFullName string) error {
	copySource := buildCopySource(c.bucketName, sourceFullName)

	_, err := c.client.CopyObject(ctx, &awss3.CopyObjectInput{
		Bucket:     aws.String(c.bucketName),
		Key:        aws.String(targetFullName),
		CopySource: aws.String(copySource),
	})
	if err != nil {
		return fmt.Errorf("s3: copy '%s' -> '%s': %w", sourceFullName, targetFullName, err)
	}

	return nil
}

func (c *Impl) composeMultipart(ctx context.Context, targetFullName string, sources []composeSource) error {
	log := c.log.GetLogger(ctx)

	mpu, err := c.client.CreateMultipartUpload(ctx, &awss3.CreateMultipartUploadInput{
		Bucket: aws.String(c.bucketName),
		Key:    aws.String(targetFullName),
	})
	if err != nil {
		return fmt.Errorf("s3: create multipart for '%s': %w", targetFullName, err)
	}

	uploadId := mpu.UploadId
	committed := false

	//nolint:contextcheck // intentional: abort uses a detached ctx so it fires even when caller canceled.
	defer func() {
		if committed {
			return
		}

		abortCtx, cancel := context.WithTimeout(context.Background(), abortMultipartTimeout)
		defer cancel()

		_, abortErr := c.client.AbortMultipartUpload(abortCtx, &awss3.AbortMultipartUploadInput{
			Bucket:   aws.String(c.bucketName),
			Key:      aws.String(targetFullName),
			UploadId: uploadId,
		})
		if abortErr != nil {
			log.WithError(abortErr).Errorf("s3: abort multipart '%s' failed", targetFullName)
		}
	}()

	completed := make([]types.CompletedPart, 0, len(sources))

	for i, src := range sources {
		partNum := int32(i + 1) //nolint:gosec // S3 caps parts at 10000; int→int32 safe.

		etag, err := c.uploadCopyPart(ctx, targetFullName, uploadId, partNum, src.fullName)
		if err != nil {
			return err
		}

		completed = append(completed, types.CompletedPart{
			PartNumber: aws.Int32(partNum),
			ETag:       etag,
		})
	}

	_, err = c.client.CompleteMultipartUpload(ctx, &awss3.CompleteMultipartUploadInput{
		Bucket:          aws.String(c.bucketName),
		Key:             aws.String(targetFullName),
		UploadId:        uploadId,
		MultipartUpload: &types.CompletedMultipartUpload{Parts: completed},
	})
	if err != nil {
		return fmt.Errorf("s3: complete multipart for '%s': %w", targetFullName, err)
	}

	committed = true

	return nil
}

func (c *Impl) uploadCopyPart(
	ctx context.Context,
	targetFullName string,
	uploadId *string,
	partNum int32,
	sourceFullName string,
) (*string, error) {
	copySource := buildCopySource(c.bucketName, sourceFullName)

	out, err := c.client.UploadPartCopy(ctx, &awss3.UploadPartCopyInput{
		Bucket:     aws.String(c.bucketName),
		Key:        aws.String(targetFullName),
		UploadId:   uploadId,
		PartNumber: aws.Int32(partNum),
		CopySource: aws.String(copySource),
	})
	if err != nil {
		return nil, fmt.Errorf("s3: upload-part-copy %d for '%s': %w", partNum, targetFullName, err)
	}

	return out.CopyPartResult.ETag, nil
}

func (c *Impl) init(ctx context.Context) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.client != nil {
		return nil
	}

	awsCfg, err := config.LoadDefaultConfig(
		ctx,
		config.WithRegion(c.region),
		config.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(c.accessKey, c.secretKey, ""),
		),
	)
	if err != nil {
		return fmt.Errorf("s3: load aws config: %w", err)
	}

	c.client = awss3.NewFromConfig(awsCfg, func(o *awss3.Options) {
		if c.endpoint != "" {
			o.BaseEndpoint = aws.String(c.endpoint)
		}

		o.UsePathStyle = c.usePathStyle
	})

	//nolint:staticcheck // SA1019: feature/s3/manager mandated by v0.6.14 patch.
	c.uploader = manager.NewUploader(c.client, func(u *manager.Uploader) {
		u.PartSize = DefaultUploadPartSize
		u.Concurrency = DefaultUploadConcurrency
	})

	return nil
}

func (c *Impl) getObjectFullName(objectName string) string {
	if c.baseDir != "" {
		return c.baseDir + "/" + objectName
	}

	return objectName
}

// buildCopySource returns the value for the x-amz-copy-source header:
// "{bucket}/{key}" with the key URL-escaped but slashes preserved, and the
// bucket/key separator left as a literal '/' so S3 can split on it.
func buildCopySource(bucket, key string) string {
	return bucket + "/" + strings.ReplaceAll(url.PathEscape(key), "%2F", "/")
}
