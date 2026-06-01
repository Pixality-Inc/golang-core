//go:build integration

package gcs

import (
	"bytes"
	"context"
	"io"
	"os"
	"testing"

	gstorage "cloud.google.com/go/storage"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/option"
)

// TestImpl_Copy_serverSide exercises the native CopierFrom path against a GCS
// emulator (fake-gcs-server). Skipped unless STORAGE_EMULATOR_HOST is set.
//
// The Google SDK still parses the credentials file even against the emulator, so
// TEST_GCS_CREDENTIALS_FILE must point at a syntactically valid (dummy) service
// account json; its contents are not verified by fake-gcs.
//
//	docker run --rm -p 4443:4443 fsouza/fake-gcs-server -scheme http -public-host localhost:4443
//	STORAGE_EMULATOR_HOST=localhost:4443 TEST_GCS_CREDENTIALS_FILE=$(pwd)/private/dummy-sa.json \
//	  go test -tags integration ./gcs/ -run TestImpl_Copy_serverSide -count=1 -v
func TestImpl_Copy_serverSide(t *testing.T) {
	if os.Getenv("STORAGE_EMULATOR_HOST") == "" {
		t.Skip("STORAGE_EMULATOR_HOST not set; skipping GCS integration test")
	}

	ctx := context.Background()
	bucket := getenvDefault("TEST_GCS_BUCKET", "golang-core-copy-test")
	credsFile := os.Getenv("TEST_GCS_CREDENTIALS_FILE")

	ensureGcsBucket(t, ctx, bucket)

	client := NewClient(credsFile, "copy-test", bucket, "base-dir", "")
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
	_, exists, err := client.FileExists(ctx, "src/object.bin")
	require.NoError(t, err)
	require.True(t, exists)
}

func ensureGcsBucket(t *testing.T, ctx context.Context, bucket string) {
	t.Helper()

	raw, err := gstorage.NewClient(ctx, option.WithoutAuthentication())
	require.NoError(t, err)

	defer func() { _ = raw.Close() }()

	// fake-gcs accepts any project id; ignore an already-existing bucket
	_ = raw.Bucket(bucket).Create(ctx, "test-project", nil)
}

func getenvDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}

	return def
}
