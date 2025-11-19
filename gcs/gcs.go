package gcs

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/pixality-inc/golang-core/logger"

	"cloud.google.com/go/storage"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

type Client interface {
	Close()

	Upload(ctx context.Context, objectName string, file io.Reader) error

	UploadFile(ctx context.Context, objectName string, filename string) error

	DeleteDir(ctx context.Context, objectName string) error

	Delete(ctx context.Context, objectName string) error

	Download(ctx context.Context, objectName string) (io.ReadCloser, error)

	DownloadFile(ctx context.Context, objectName string, filename string) error

	FileExists(ctx context.Context, objectName string) (*storage.ObjectAttrs, bool, error)

	Compose(ctx context.Context, objectName string, chunks []string) error

	GetPublicUrl(ctx context.Context, objectName string) (string, error)
}

type Impl struct {
	log                 logger.Loggable
	credentialsFilename string
	name                string
	bucketName          string
	baseDir             string
	basePublicUrl       string
	client              *storage.Client
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

	writer := c.client.Bucket(c.bucketName).Object(objectFullName).NewWriter(ctx)

	defer func() {
		err := writer.Close()
		if err != nil {
			log.WithError(err).Errorf("failed to close writer for '%s'", objectFullName)
		}
	}()

	if _, err := io.Copy(writer, file); err != nil {
		return err
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

	it := bucket.Objects(ctx, &storage.Query{Prefix: objectFullName})

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

func (c *Impl) FileExists(ctx context.Context, objectName string) (*storage.ObjectAttrs, bool, error) {
	if err := c.init(ctx); err != nil {
		return nil, false, err
	}

	objectFullName := c.getObjectFullName(objectName)

	attrs, err := c.client.Bucket(c.bucketName).Object(objectFullName).Attrs(ctx)
	if err != nil && errors.Is(err, storage.ErrObjectNotExist) {
		return nil, false, nil
	} else if err != nil {
		return nil, false, err
	}

	return attrs, true, nil
}

func (c *Impl) Compose(ctx context.Context, objectName string, chunks []string) error {
	c.log.GetLogger(ctx).Infof("Composing object '%s' from %d chunks", objectName, len(chunks))

	if err := c.init(ctx); err != nil {
		return err
	}

	bucket := c.client.Bucket(c.bucketName)

	objectFullName := c.getObjectFullName(objectName)

	chunkObjects := make([]*storage.ObjectHandle, len(chunks))

	for n, chunk := range chunks {
		chunkObjectFullName := c.getObjectFullName(chunk)
		chunkObjects[n] = bucket.Object(chunkObjectFullName)
	}

	_, err := bucket.Object(objectFullName).ComposerFrom(chunkObjects...).Run(ctx)
	if err != nil {
		return err
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

func (c *Impl) init(ctx context.Context) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.client != nil {
		return nil
	}

	client, err := storage.NewClient(ctx, option.WithCredentialsFile(c.credentialsFilename))
	if err != nil {
		return err
	}

	c.client = client

	return nil
}

func (c *Impl) getObjectFullName(objectName string) string {
	if c.baseDir != "" {
		return c.baseDir + "/" + objectName
	} else {
		return objectName
	}
}
