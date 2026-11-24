//go:build integration

package s3

import (
	"bytes"
	"context"
	"io"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/stretchr/testify/require"
)

// TestImpl_Upload_byLength sends objects of every shape Upload tells apart to a real S3-compatible
// backend and reads them back: short and long streams, a partly read file, an object streamed
// from another client and a gzip object, whose stored bytes and Content-Encoding must survive.
// skipped unless TEST_S3_ENDPOINT is set, see TestImpl_Copy_serverSide for a local backend.
// TEST_S3_FORCE_GZIP_ACCEPT_ENCODING=false runs it the way the clients of r2 are set up. the objects
// go under a prefix of the run and are removed at the end.
func TestImpl_Upload_byLength(t *testing.T) {
	t.Parallel()

	endpoint := os.Getenv("TEST_S3_ENDPOINT")
	if endpoint == "" {
		t.Skip("TEST_S3_ENDPOINT not set; skipping S3 integration test")
	}

	ctx := context.Background()
	accessKey := os.Getenv("TEST_S3_ACCESS_KEY")
	secretKey := os.Getenv("TEST_S3_SECRET_KEY")
	bucket := getenvDefault("TEST_S3_BUCKET", "golang-core-upload-test")
	region := getenvDefault("TEST_S3_REGION", "us-east-1")

	ensureS3Bucket(t, ctx, endpoint, accessKey, secretKey, bucket)

	var opts []Option
	if os.Getenv("TEST_S3_FORCE_GZIP_ACCEPT_ENCODING") == "false" {
		opts = append(opts, WithForceGzipAcceptEncoding(false))
	}

	run := "golang-core-upload-test/" + strconv.FormatInt(time.Now().UnixNano(), 10)
	srcDir, dstDir := run+"/src", run+"/dst"

	client := NewClient("upload-test", endpoint, region, accessKey, secretKey, bucket, srcDir, "", true, opts...)
	other := NewClient("upload-test-dst", endpoint, region, accessKey, secretKey, bucket, dstDir, "", true, opts...)

	t.Cleanup(func() {
		for _, c := range []Client{client, other} {
			if err := c.DeleteDir(context.Background(), ""); err != nil {
				t.Errorf("remove the objects of the run: %v", err)
			}

			c.Close()
		}
	})

	random := func(t *testing.T, size int) []byte {
		t.Helper()

		data := make([]byte, size)

		_, err := rand.NewChaCha8([32]byte{7}).Read(data)
		require.NoError(t, err)

		return data
	}

	download := func(t *testing.T, c Client, name string) []byte {
		t.Helper()

		reader, err := c.Download(ctx, name)
		require.NoError(t, err)

		defer func() { _ = reader.Close() }()

		var buf bytes.Buffer

		_, err = io.Copy(&buf, reader)
		require.NoError(t, err)

		return buf.Bytes()
	}

	t.Run("short stream", func(t *testing.T) {
		t.Parallel()

		data := random(t, 100*1024)
		require.NoError(t, client.Upload(ctx, "short.bin", plainReader{bytes.NewReader(data)}))
		require.Equal(t, data, download(t, client, "short.bin"))
	})

	t.Run("long stream in parallel parts", func(t *testing.T) {
		t.Parallel()

		data := random(t, int(3*DefaultUploadPartSize)+12345)
		require.NoError(t, client.Upload(ctx, "long.bin", plainReader{bytes.NewReader(data)}))
		require.Equal(t, data, download(t, client, "long.bin"))
	})

	t.Run("long reader of a known length", func(t *testing.T) {
		t.Parallel()

		data := random(t, int(2*DefaultUploadPartSize)+7)
		require.NoError(t, client.Upload(ctx, "long-known.bin", bytes.NewReader(data)))
		require.Equal(t, data, download(t, client, "long-known.bin"))
	})

	t.Run("partly read file", func(t *testing.T) {
		t.Parallel()

		data := random(t, 5*1024*1024)
		filename := filepath.Join(t.TempDir(), "object.bin")
		require.NoError(t, os.WriteFile(filename, data, 0o600))

		file, err := os.Open(filename)
		require.NoError(t, err)
		t.Cleanup(func() { _ = file.Close() })

		_, err = file.Seek(1000, io.SeekStart)
		require.NoError(t, err)

		require.NoError(t, client.Upload(ctx, "partly.bin", file))
		require.Equal(t, data[1000:], download(t, client, "partly.bin"))
	})

	for name, size := range map[string]int{
		"small object streamed from another client": 2*1024*1024 + 1,
		"large object streamed from another client": int(DefaultUploadPartSize)*3 + 4096,
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			key := strings.ReplaceAll(name, " ", "-") + ".bin"
			data := random(t, size)
			require.NoError(t, client.Upload(ctx, key, bytes.NewReader(data)))

			reader, err := client.Download(ctx, key)
			require.NoError(t, err)

			defer func() { _ = reader.Close() }()

			require.NoError(t, other.Upload(ctx, key, reader))
			require.Equal(t, data, download(t, other, key))
		})
	}

	// the encoding comes from the .gz name on both sides, the bytes must come over unpacked by nobody
	t.Run("gzip object keeps its bytes and encoding", func(t *testing.T) {
		t.Parallel()

		data := random(t, 2*1024*1024)
		require.NoError(t, client.Upload(ctx, "normals.tar.gz", bytes.NewReader(data)))

		reader, err := client.Download(ctx, "normals.tar.gz")
		require.NoError(t, err)

		defer func() { _ = reader.Close() }()

		require.NoError(t, other.Upload(ctx, "normals.tar.gz", reader))
		require.Equal(t, data, download(t, other, "normals.tar.gz"))

		host, secure, err := parseEndpoint(endpoint)
		require.NoError(t, err)

		raw, err := minio.New(host, &minio.Options{Creds: credentials.NewStaticV4(accessKey, secretKey, ""), Secure: secure})
		require.NoError(t, err)

		info, err := raw.StatObject(ctx, bucket, dstDir+"/normals.tar.gz", minio.StatObjectOptions{})
		require.NoError(t, err)
		require.Equal(t, "gzip", info.Metadata.Get("Content-Encoding"))
		require.Equal(t, int64(len(data)), info.Size)
	})
}
