package s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

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

// MaxSingleCopySize is the maximum object size for a single server-side PUT-copy (CopyObject). S3
// caps the copy source at 5 GiB; larger sources must be copied with a server-side multipart copy
// (ComposeObject -> UploadPartCopy).
const MaxSingleCopySize int64 = 5 * 1024 * 1024 * 1024

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

// ErrInvalidEndpoint is returned by init when the caller passed an endpoint
// minio-go cannot use directly: unsupported scheme, or a URL carrying a
// path / query / fragment that we would otherwise have to silently drop.
var ErrInvalidEndpoint = errors.New("s3: invalid endpoint")

type Client interface {
	Close()

	Upload(ctx context.Context, objectName string, file io.Reader) error
	UploadFile(ctx context.Context, objectName string, filename string) error

	DeleteDir(ctx context.Context, objectName string) error
	Delete(ctx context.Context, objectName string) error

	Copy(ctx context.Context, srcObjectName string, dstObjectName string) error

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

func (c *Impl) Copy(ctx context.Context, srcObjectName string, dstObjectName string) error {
	c.log.GetLogger(ctx).Infof("Copying object '%s' to '%s'", srcObjectName, dstObjectName)

	if err := c.init(ctx); err != nil {
		return err
	}

	srcFullName := c.getObjectFullName(srcObjectName)
	dstFullName := c.getObjectFullName(dstObjectName)

	src := minio.CopySrcOptions{Bucket: c.bucketName, Object: srcFullName}
	dst := minio.CopyDestOptions{Bucket: c.bucketName, Object: dstFullName}

	// CopyObject is a single server-side PUT-copy: it duplicates the object and all its metadata
	// server-side, but S3 caps the copy source at MaxSingleCopySize (5 GiB). This is the hot path and
	// stays free of any extra request for the common (<= 5 GiB) case.
	if _, err := c.client.CopyObject(ctx, dst, src); err == nil {
		return nil
	} else if composeErr := c.copyLarge(ctx, srcFullName, dstFullName, err); composeErr != nil {
		return composeErr
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
	// listErrCh is buffered with cap 1 and written via a defer so we always
	// signal the receiver below, even if ctx is canceled or RemoveObjects
	// stops draining toDelete mid-flight (which would otherwise leak this
	// goroutine on a blocked send).
	toDelete := make(chan minio.ObjectInfo)
	listErrCh := make(chan error, 1)

	go func() {
		var listErr error

		defer func() {
			close(toDelete)

			listErrCh <- listErr
		}()

		for info := range listCh {
			if info.Err != nil {
				listErr = info.Err

				return
			}

			select {
			case toDelete <- info:
			case <-ctx.Done():
				listErr = ctx.Err()

				return
			}
		}
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

	// Force Accept-Encoding: gzip on the request. Some S3-compatible backends
	// (observed empirically on the staging bucket) perform decompressive
	// transcoding when the client does NOT advertise an encoding: an object
	// stored with Content-Encoding: gzip gets unpacked server-side before the
	// bytes reach us. Sending Accept-Encoding: gzip tells the server "give me
	// the stored bytes as-is". Pairs with DisableCompression: true on the
	// transport (set in init) which keeps net/http from auto-decompressing on
	// the way back. This is the S3 equivalent of GCS's ReadCompressed(true).
	opts := minio.GetObjectOptions{}
	opts.Set("Accept-Encoding", "gzip")

	obj, err := c.client.GetObject(ctx, c.bucketName, objectFullName, opts)
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
// minio.ErrorResponse via ToErrorResponse — we accept NoSuchKey (S3 spec
// code), NotFound (returned by some providers / for HEAD without body),
// and a raw HTTP 404 status as the catch-all.
func isNotFoundErr(err error) bool {
	if err == nil {
		return false
	}

	resp := minio.ToErrorResponse(err)

	return resp.Code == "NoSuchKey" ||
		resp.Code == "NotFound" ||
		resp.StatusCode == http.StatusNotFound
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

const (
	// lastModifiedHeader is the response header minio-go strictly parses into
	// ObjectInfo.LastModified via ToObjectInfo.
	lastModifiedHeader = "Last-Modified"

	// placeholderLastModified is a valid RFC1123 timestamp that minio-go can
	// parse. The concrete instant is irrelevant: it only stands in for responses
	// that carry no Last-Modified header and whose modification time is never
	// read by this package.
	placeholderLastModified = "Thu, 01 Jan 1970 00:00:00 GMT"
)

// lastModifiedFallbackTransport works around S3-compatible backends that omit
// the Last-Modified header on some responses (observed on SeaweedFS GET
// replies, whose HEAD replies do carry it). minio-go treats an absent
// Last-Modified as a hard error in ToObjectInfo, which aborts an otherwise
// successful read mid-stream once the body is already being copied. No caller
// in this package consumes the modification time parsed from a GET response, so
// a constant placeholder is injected only for GET responses that lack the
// header.
//
// The fallback is deliberately scoped to GET: HEAD/Stat is the path through
// which a real modification time could ever reach a caller (e.g. ReadDir reads
// it, though from the ListObjects XML body rather than these headers), so those
// responses are left untouched and a genuinely missing Last-Modified there
// stays a loud error instead of being masked by the placeholder.
type lastModifiedFallbackTransport struct {
	base http.RoundTripper
}

func (t lastModifiedFallbackTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return resp, err
	}

	if req.Method == http.MethodGet && resp.Header.Get(lastModifiedHeader) == "" {
		resp.Header.Set(lastModifiedHeader, placeholderLastModified)
	}

	return resp, nil
}

// sourceResponseHeaderTimeout bounds how long a request waits for the backend
// to start sending its response headers. It restores the value minio-go uses in
// its own DefaultTransport (time.Minute); the custom transport below, built for
// DisableCompression, would otherwise leave this unset (0 = wait forever).
// SeaweedFS was observed to intermittently stall a GET before the first response
// byte and never recover on that connection, which without a bound blocks the
// caller indefinitely; the timeout turns that into an error that minio-go
// retries, usually landing on a healthy window.
//
// Scope: this bounds only the time to response headers. The clock starts after
// the request body is fully written, so it caps neither a slow upload body nor
// the streaming read of the response body. Server-side operations that return
// headers promptly and then stream (CopyObject sends 200 then keep-alive
// whitespace) are unaffected. Matching minio-go's own default keeps every
// operation that works under a vanilla minio client working here too.
const sourceResponseHeaderTimeout = time.Minute

// newS3Transport builds the *http.Transport shared by the minio client.
//
// DisableCompression is set so net/http neither advertises Accept-Encoding: gzip
// nor transparently gunzips responses. Objects with Content-Encoding: gzip (e.g.
// .csv.gz served to browsers) must reach our backends byte-for-byte; the header
// is a contract with the frontend, not an instruction to our backends.
// Constructing fresh here (instead of cloning http.DefaultTransport) keeps us
// independent of process-wide transport wrappers (otelhttp, datadog, etc.).
//
// ResponseHeaderTimeout is honored on HTTP/1 connections, which is what the
// affected backend (SeaweedFS) uses; net/http ignores it on HTTP/2, exactly as
// minio-go's own DefaultTransport does. Healthy backends that negotiate HTTP/2
// (e.g. Hetzner) were never the stalling backend, so this is not a regression.
func newS3Transport() *http.Transport {
	return &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          256,
		MaxIdleConnsPerHost:   16,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableCompression:    true,
		ResponseHeaderTimeout: sourceResponseHeaderTimeout,
	}
}

// copyLarge handles the fallback for an object that CopyObject could not copy in one PUT, which is
// expected when the source is over the 5 GiB single-copy limit. It confirms the source really is over
// the limit (otherwise the original copy error is surfaced unchanged) and then copies it with a
// server-side multipart copy (ComposeObject -> UploadPartCopy by byte range, up to ~5 TiB). The
// multipart path does not carry the source content metadata, so it is reproduced on the destination
// explicitly.
func (c *Impl) copyLarge(ctx context.Context, srcFullName, dstFullName string, copyErr error) error {
	info, statErr := c.client.StatObject(ctx, c.bucketName, srcFullName, minio.StatObjectOptions{})
	if statErr != nil || info.Size <= MaxSingleCopySize {
		// not provably an over-the-limit source: surface the original copy error
		return fmt.Errorf("s3: copy '%s' to '%s': %w", srcFullName, dstFullName, copyErr)
	}

	src := minio.CopySrcOptions{Bucket: c.bucketName, Object: srcFullName}
	dst := minio.CopyDestOptions{
		Bucket:          c.bucketName,
		Object:          dstFullName,
		ReplaceMetadata: true,
		UserMetadata:    largeCopyMetadata(info),
	}

	if _, err := c.client.ComposeObject(ctx, dst, src); err != nil {
		return fmt.Errorf("s3: multipart copy '%s' to '%s': %w", srcFullName, dstFullName, err)
	}

	return nil
}

// largeCopyMetadata reproduces the source object content metadata for a multipart copy. The
// ComposeObject multipart path does not carry it over, so the source content-type, the other content
// headers and the user metadata are passed explicitly on the destination. Keys that name a standard
// HTTP header (content-type, cache-control, ...) are applied as-is by the SDK; the rest become
// x-amz-meta-* entries. User tags are preserved by ComposeObject itself (ReplaceTags stays false).
func largeCopyMetadata(info minio.ObjectInfo) map[string]string {
	meta := make(map[string]string, len(info.UserMetadata)+5)

	maps.Copy(meta, info.UserMetadata)

	if info.ContentType != "" {
		meta["Content-Type"] = info.ContentType
	}

	for _, h := range []string{"Content-Encoding", "Content-Disposition", "Content-Language", "Cache-Control"} {
		if v := info.Metadata.Get(h); v != "" {
			meta[h] = v
		}
	}

	return meta
}

// init is intentionally context-agnostic — minio.New does not perform a
// round-trip, so there is nothing to cancel here. The ctx parameter is kept
// on the signature only so call sites stay symmetric with the gcs sibling.
func (c *Impl) init(_ context.Context) error {
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
		Creds:     credentials.NewStaticV4(c.accessKey, c.secretKey, ""),
		Secure:    secure,
		Region:    c.region,
		Transport: lastModifiedFallbackTransport{base: newS3Transport()},
	}

	if c.usePathStyle {
		opts.BucketLookup = minio.BucketLookupPath
	}

	client, err := minio.New(host, opts)
	if err != nil {
		return fmt.Errorf("s3: init minio client: %w", err)
	}

	c.client = client

	return nil
}

// parseEndpoint splits a caller-supplied endpoint into a host (no scheme) plus
// a Secure flag, since minio.New takes them separately. We reject anything
// the caller might have plausibly meant differently than what minio-go will
// do with it: empty endpoint, non-http(s) scheme, or a URL with a path /
// query / fragment that minio-go would silently ignore.
func parseEndpoint(endpoint string) (string, bool, error) {
	if endpoint == "" {
		return "", false, ErrEmptyEndpoint
	}

	parsed, err := url.Parse(endpoint)
	if err != nil {
		return "", false, fmt.Errorf("s3: parse endpoint %q: %w", endpoint, err)
	}

	if parsed.Host == "" {
		// No scheme; the whole string is treated as a bare host. Disallow
		// anything that looks like a URL fragment we'd otherwise drop.
		if strings.ContainsAny(endpoint, "/?#") {
			return "", false, fmt.Errorf("%w: bare host must not contain path/query/fragment: %q", ErrInvalidEndpoint, endpoint)
		}

		return endpoint, true, nil
	}

	switch parsed.Scheme {
	case "http", "https":
	default:
		return "", false, fmt.Errorf("%w: scheme must be http or https, got %q: %q", ErrInvalidEndpoint, parsed.Scheme, endpoint)
	}

	if parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", false, fmt.Errorf("%w: must not contain path/query/fragment: %q", ErrInvalidEndpoint, endpoint)
	}

	return parsed.Host, parsed.Scheme == "https", nil
}

func (c *Impl) getObjectFullName(objectName string) string {
	if c.baseDir != "" {
		return c.baseDir + "/" + objectName
	}

	return objectName
}
