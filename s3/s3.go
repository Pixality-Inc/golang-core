package s3

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
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

// DefaultUploadPartSize is the part size Upload uses for an object larger than it. Upload raises it
// for an object of a known length that does not fit maxUploadParts parts of it; a stream of an
// unknown length fits maxUploadParts parts, 156 GiB, a longer one fails with ErrUploadStreamTooLong.
const DefaultUploadPartSize int64 = 16 * 1024 * 1024

// DefaultUploadConcurrency is the most parts Upload sends at once. measured on r2 with 1 GiB:
// 16 MiB x 8 99 MiB/s, 16 MiB x 4 53, 64 MiB x 4 64, parts one by one 15-32. for a body read in
// order (a stream, a minio object) minio-go takes one buffer of threads x part size up front, 128 MiB
// with the defaults, a stream also keeps up to uploadPeekSize read ahead until its first part is
// read; a file or another ReaderAt of a known length is sent in sections of itself without buffers.
const DefaultUploadConcurrency = 8

// maxUploadParts is the s3 limit of parts in one multipart upload
const maxUploadParts int64 = 10000

// uploadPeekSize is how much of a stream of unknown length Upload reads ahead: a stream that ends
// within it goes as one PUT of a known length, a longer one as a multipart upload
const uploadPeekSize = DefaultUploadPartSize

// readAheadSmallSize is how much of a stream is read ahead into a growing buffer before the buffer
// for the whole peek is taken
const readAheadSmallSize = 1024 * 1024

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

// ErrUploadSizeMismatch is returned by Upload when the object came out of another size than the
// source said it had; the object is removed.
var ErrUploadSizeMismatch = errors.New("s3: uploaded size does not match the source")

// ErrUploadStreamTooLong is returned by Upload when a stream of an unknown length did not fit
// maxUploadParts parts: minio-go completes the object without the rest, so it is removed.
var ErrUploadStreamTooLong = errors.New("s3: stream is longer than a multipart upload takes")

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
	Stat(ctx context.Context, objectName string) (fs.FileInfo, error)

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

	// responseHeaderTimeout is the transport ResponseHeaderTimeout. NewClient
	// seeds it with sourceResponseHeaderTimeout and WithResponseHeaderTimeout
	// overrides it.
	responseHeaderTimeout time.Duration

	// forceGzipAcceptEncoding makes Download send Accept-Encoding: gzip. NewClient
	// seeds it with true and WithForceGzipAcceptEncoding overrides it.
	forceGzipAcceptEncoding bool

	client *minio.Client
	mutex  sync.Mutex
}

// Option configures a Client at construction time.
type Option func(*Impl)

// WithResponseHeaderTimeout overrides how long a request waits for the backend
// to start sending its response headers (http.Transport.ResponseHeaderTimeout).
// default is sourceResponseHeaderTimeout. it is worth raising for backends that
// are legitimately slow to first byte, e.g. seaweedfs locating a cold object.
// honored on http/1 only, net/http ignores it on http/2 same as minio-go's own
// default transport.
func WithResponseHeaderTimeout(timeout time.Duration) Option {
	return func(c *Impl) {
		c.responseHeaderTimeout = timeout
	}
}

// WithForceGzipAcceptEncoding controls whether Download sends Accept-Encoding: gzip.
// default is true: seaweedfs unpacks objects stored with Content-Encoding: gzip
// unless the client advertises gzip. it is turned off for backends that compress
// on the fly instead, e.g. cloudflare r2: asked for gzip it compresses plain
// json/txt objects and weakens their etag to W/"...", minio-go then sends that
// weak etag in If-Match on the follow-up GET and r2 answers 412 every time.
// r2 returns objects stored with Content-Encoding: gzip as is without the header.
func WithForceGzipAcceptEncoding(force bool) Option {
	return func(c *Impl) {
		c.forceGzipAcceptEncoding = force
	}
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
	opts ...Option,
) Client {
	client := &Impl{
		log: logger.NewLoggableImplWithServiceAndFields(
			"s3",
			logger.Fields{
				"name":   name,
				"bucket": bucketName,
			},
		),
		name:                    name,
		bucketName:              bucketName,
		baseDir:                 baseDir,
		basePublicUrl:           basePublicUrl,
		endpoint:                endpoint,
		region:                  region,
		accessKey:               accessKey,
		secretKey:               secretKey,
		usePathStyle:            usePathStyle,
		responseHeaderTimeout:   sourceResponseHeaderTimeout,
		forceGzipAcceptEncoding: true,
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
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

	source, err := newUploadSource(file)
	if err != nil {
		return fmt.Errorf("s3: upload '%s': %w", objectFullName, err)
	}

	partSize := uploadPartSize(source.size)

	opts := minio.PutObjectOptions{
		ContentType:     metadata.ContentType(),
		ContentEncoding: metadata.ContentEncoding(),
		PartSize:        uint64(partSize), //nolint:gosec // uploadPartSize is never below DefaultUploadPartSize
		NumThreads:      uploadThreads(source.size, partSize),
		// a body minio-go can read only in order goes in parallel parts only with this, through
		// buffers; a ReaderAt of a known length is sent in parallel sections of itself without it.
		// it must not meet TrailingHeaders or a checksum option: in minio-go v7.1.0 the parts of
		// this path share one hasher.
		ConcurrentStreamParts: source.inOrder,
	}

	info, err := c.client.PutObject(ctx, c.bucketName, objectFullName, source.body, source.size, opts)
	if err != nil {
		return fmt.Errorf("s3: upload '%s': %w", objectFullName, err)
	}

	if err := source.check(info.Size); err != nil {
		if removeErr := c.client.RemoveObject(ctx, c.bucketName, objectFullName, minio.RemoveObjectOptions{}); removeErr != nil {
			log.WithError(removeErr).Errorf("failed to remove the incomplete object '%s'", objectFullName)
		}

		return fmt.Errorf("s3: upload '%s': %w", objectFullName, err)
	}

	return nil
}

// uploadPartSize is DefaultUploadPartSize, raised for an object of a known length that does not
// fit maxUploadParts parts of it
func uploadPartSize(size int64) int64 {
	if size > DefaultUploadPartSize*maxUploadParts {
		return (size + maxUploadParts - 1) / maxUploadParts
	}

	return DefaultUploadPartSize
}

// uploadThreads is DefaultUploadConcurrency, cut to the number of parts of an object of a known
// length: a 17 MiB object read in order takes buffers for its 2 parts, not for 8
func uploadThreads(size, partSize int64) uint {
	if size < 0 {
		return DefaultUploadConcurrency
	}

	parts := (size + partSize - 1) / partSize

	return uint(max(1, min(parts, DefaultUploadConcurrency)))
}

// uploadSource is the body Upload hands to minio-go. minio-go sends an object of a known length up
// to the part size in one PUT, of an unknown length always as a multipart upload (create, part,
// complete) with a part size buffer, which made small objects about 1.7 times slower on r2. so the
// length is taken from the readers that can tell it and a stream is read ahead up to
// uploadPeekSize to learn whether it is small.
type uploadSource struct {
	body io.Reader
	// size is -1 while the length is unknown
	size int64
	// inOrder marks a body minio-go can read only from its start to its end
	inOrder bool
}

func newUploadSource(file io.Reader) (uploadSource, error) {
	if source, ok := sizedSource(file); ok {
		return source, nil
	}

	// one byte over the peek tells a stream of exactly uploadPeekSize from a longer one
	head, ended, err := readAhead(file, uploadPeekSize+1)
	if err != nil {
		return uploadSource{}, fmt.Errorf("read ahead: %w", err)
	}

	if ended {
		return uploadSource{body: bytes.NewReader(head), size: int64(len(head))}, nil
	}

	return uploadSource{body: io.MultiReader(bytes.NewReader(head), cutTailReader{file}), size: -1, inOrder: true}, nil
}

// cutTailReader keeps a cut stream an error for minio-go: it reads the parts of a stream with a
// readFull that takes io.ErrUnexpectedEOF for a short last part and completes the object without
// the rest, while a wrapped one aborts the upload
type cutTailReader struct {
	io.Reader
}

func (r cutTailReader) Read(p []byte) (int, error) {
	n, err := r.Reader.Read(p)
	if errors.Is(err, io.ErrUnexpectedEOF) {
		return n, fmt.Errorf("stream cut: %w", err)
	}

	return n, err //nolint:wrapcheck // io.EOF must stay as is
}

// check compares what minio-go reports as uploaded with the source: a known length must match, a
// stream must be drained, minio-go stops a stream after maxUploadParts parts and completes the
// object without the rest
func (s uploadSource) check(uploaded int64) error {
	if s.size >= 0 {
		if uploaded != s.size {
			return fmt.Errorf("%w: %d bytes of %d", ErrUploadSizeMismatch, uploaded, s.size)
		}

		return nil
	}

	var probe [1]byte

	n, err := io.ReadFull(s.body, probe[:])
	if n > 0 {
		return fmt.Errorf("%w: more than %d bytes", ErrUploadStreamTooLong, uploaded)
	}

	if err != nil && !errors.Is(err, io.EOF) {
		return fmt.Errorf("check the end of the stream: %w", err)
	}

	return nil
}

// sizedSource takes the length of a reader that can tell it. minio-go retries a request by seeking
// its body to 0 and reads a ReaderAt at absolute offsets, so a reader already read past its start
// is cut to a section that begins at the current offset.
func sizedSource(file io.Reader) (uploadSource, bool) {
	if object, ok := file.(*minio.Object); ok {
		return objectSource(object)
	}

	readerAt, isReaderAt := file.(io.ReaderAt)
	seeker, isSeeker := file.(io.Seeker)

	if !isReaderAt || !isSeeker {
		return uploadSource{}, false
	}

	total, ok := readerSize(file)
	if !ok {
		return uploadSource{}, false
	}

	offset, err := seeker.Seek(0, io.SeekCurrent)
	if err != nil || offset < 0 || offset > total {
		return uploadSource{}, false
	}

	if offset == 0 {
		return uploadSource{body: file, size: total}, true
	}

	return uploadSource{body: io.NewSectionReader(readerAt, offset, total-offset), size: total - offset}, true
}

// objectSource hands a minio object to minio-go as the *minio.Object itself: minio-go reads that in
// order with one GET, while anything else that is a ReaderAt it reads by ReadAt, and every ReadAt of
// an object is a ranged GET of its own. the length is the Content-Length of the HEAD Download sent
// with the same Accept-Encoding as the GET. an object read past its start goes the way of a stream.
func objectSource(object *minio.Object) (uploadSource, bool) {
	info, err := object.Stat()
	if err != nil || info.Size < 0 {
		return uploadSource{}, false
	}

	offset, err := object.Seek(0, io.SeekCurrent)
	if err != nil || offset != 0 {
		return uploadSource{}, false
	}

	return uploadSource{body: object, size: info.Size, inOrder: true}, true
}

// readerSize is the whole length of the readers that report it: bytes.Reader, strings.Reader and
// io.SectionReader by Size, a regular os.File by its stat
func readerSize(file io.Reader) (int64, bool) {
	switch reader := file.(type) {
	case interface{ Size() int64 }:
		return reader.Size(), true

	case *os.File:
		info, err := reader.Stat()
		if err != nil || !info.Mode().IsRegular() {
			return 0, false
		}

		return info.Size(), true
	}

	return 0, false
}

// readAhead reads a stream up to limit bytes and reports whether it ended within them. the first
// readAheadSmallSize go through a growing buffer, so a small stream costs about its own size; a
// longer one is read into one buffer of limit bytes instead of a buffer doubling up to twice of it.
// only io.EOF ends a stream: io.ErrUnexpectedEOF of a source is a cut body (net/http reports so a
// response shorter than its Content-Length) and fails the upload.
func readAhead(file io.Reader, limit int64) ([]byte, bool, error) {
	var small bytes.Buffer

	// one byte over the small size tells a stream of exactly that size from a longer one
	smallLimit := min(readAheadSmallSize+1, limit)

	read, err := io.CopyN(&small, file, smallLimit)
	if errors.Is(err, io.EOF) {
		return small.Bytes(), true, nil
	} else if err != nil {
		return nil, false, err
	}

	if read == limit {
		return small.Bytes(), false, nil
	}

	head := make([]byte, limit)
	copy(head, small.Bytes())

	for read < limit {
		n, err := file.Read(head[read:])
		read += int64(n)

		if errors.Is(err, io.EOF) {
			return head[:read], true, nil
		} else if err != nil {
			return nil, false, err
		}
	}

	return head, false, nil
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
	// Backends that compress on the fly opt out via WithForceGzipAcceptEncoding.
	opts := minio.GetObjectOptions{}
	if c.forceGzipAcceptEncoding {
		opts.Set("Accept-Encoding", "gzip")
	}

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

// isNotFoundErr reports whether err represents a 404 for an S3 object. The codes that name a
// missing key live in storage.IsNotFound, shared with the other providers; the raw 404 status
// is read as not-found only here, because the callers below ask about one named object (a
// HEAD), where a 404 without a body can mean nothing else. On a listing minio-go labels the
// same bodyless 404 as NoSuchBucket, which is a failure rather than an empty folder.
func isNotFoundErr(err error) bool {
	if err == nil {
		return false
	}

	if storage.IsNotFound(err) {
		return true
	}

	return minio.ToErrorResponse(err).StatusCode == http.StatusNotFound
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
	log  logger.Loggable
}

func (t lastModifiedFallbackTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	resp, err := t.base.RoundTrip(req)
	if err != nil {
		return resp, err
	}

	// seaweedfs answers 503 + Retry-After while it warms a cold object from
	// origin. log the response headers so the Retry-After it actually sends is
	// observable, its value is not seconds and can be minutes
	if resp.StatusCode == http.StatusServiceUnavailable {
		t.log.GetLoggerWithoutContext().
			WithFields(logger.Fields{
				"method":      req.Method,
				"url":         req.URL.String(),
				"status":      resp.StatusCode,
				"retry_after": resp.Header.Get("Retry-After"),
				"headers":     resp.Header,
			}).
			Warn("s3 backend returned 503")
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
func newS3Transport(responseHeaderTimeout time.Duration) *http.Transport {
	return &http.Transport{
		Proxy:                 http.ProxyFromEnvironment,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          256,
		MaxIdleConnsPerHost:   16,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableCompression:    true,
		ResponseHeaderTimeout: responseHeaderTimeout,
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
	meta := make(map[string]string, len(info.UserMetadata)+6)

	maps.Copy(meta, info.UserMetadata)

	if info.ContentType != "" {
		meta["Content-Type"] = info.ContentType
	}

	for _, h := range []string{"Content-Encoding", "Content-Disposition", "Content-Language", "Cache-Control"} {
		if v := info.Metadata.Get(h); v != "" {
			meta[h] = v
		}
	}

	// Expires is a standard content header that the single-PUT CopyObject path preserves, but
	// minio-go parses it into ObjectInfo.Expires (a time.Time) rather than leaving it in Metadata,
	// so it has to be reproduced from that field. The SDK treats "Expires" as a standard header on
	// write, and http.TimeFormat matches the encoding minio-go uses for it on PutObject.
	if !info.Expires.IsZero() {
		meta["Expires"] = info.Expires.UTC().Format(http.TimeFormat)
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
		Transport: lastModifiedFallbackTransport{base: newS3Transport(c.responseHeaderTimeout), log: c.log},
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
