//go:build integration

package s3

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/stretchr/testify/require"
)

// TestImpl_Copy_serverSide exercises the native minio CopyObject path against a
// real S3-compatible backend. Skipped unless TEST_S3_ENDPOINT is set.
//
// Run a throwaway MinIO and the test:
//
//	docker run --rm -p 9000:9000 -e MINIO_ROOT_USER=minioadmin -e MINIO_ROOT_PASSWORD=minioadmin \
//	  minio/minio server /data
//	TEST_S3_ENDPOINT=http://localhost:9000 TEST_S3_ACCESS_KEY=minioadmin TEST_S3_SECRET_KEY=minioadmin \
//	  go test -tags integration ./s3/ -run TestImpl_Copy_serverSide -count=1 -v
func TestImpl_Copy_serverSide(t *testing.T) {
	endpoint := os.Getenv("TEST_S3_ENDPOINT")
	if endpoint == "" {
		t.Skip("TEST_S3_ENDPOINT not set; skipping S3 integration test")
	}

	ctx := context.Background()
	accessKey := os.Getenv("TEST_S3_ACCESS_KEY")
	secretKey := os.Getenv("TEST_S3_SECRET_KEY")
	bucket := getenvDefault("TEST_S3_BUCKET", "golang-core-copy-test")
	region := getenvDefault("TEST_S3_REGION", "us-east-1")

	ensureS3Bucket(t, ctx, endpoint, accessKey, secretKey, bucket)

	client := NewClient("copy-test", endpoint, region, accessKey, secretKey, bucket, "base-dir", "", true)
	t.Cleanup(client.Close)

	want := []byte("server-side-copy-payload")
	require.NoError(t, client.Upload(ctx, "src/object.bin", bytes.NewReader(want)))

	require.NoError(t, client.Copy(ctx, "src/object.bin", "dst/copy.bin"))

	// destination carries identical content
	rc, err := client.Download(ctx, "dst/copy.bin")
	require.NoError(t, err)

	defer func() { _ = rc.Close() }()

	got, err := io.ReadAll(rc)
	require.NoError(t, err)
	require.Equal(t, want, got)

	// source is preserved after a copy
	exists, err := client.FileExists(ctx, "src/object.bin")
	require.NoError(t, err)
	require.True(t, exists)
}

func ensureS3Bucket(t *testing.T, ctx context.Context, endpoint, accessKey, secretKey, bucket string) {
	t.Helper()

	host, secure, err := parseEndpoint(endpoint)
	require.NoError(t, err)

	raw, err := minio.New(host, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretKey, ""),
		Secure: secure,
	})
	require.NoError(t, err)

	exists, err := raw.BucketExists(ctx, bucket)
	require.NoError(t, err)

	if !exists {
		require.NoError(t, raw.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}))
	}
}

func getenvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	return def
}
