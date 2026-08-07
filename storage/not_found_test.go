package storage

import (
	"errors"
	"fmt"
	"io/fs"
	"net/http"
	"testing"

	gcs "cloud.google.com/go/storage"
	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/require"
)

var errBackendDown = errors.New("backend down")

func TestIsNotFound(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil",
			err:  nil,
			want: false,
		},
		{
			name: "os provider, wrapped",
			err:  fmt.Errorf("storage.ReadFile(%s): %w", "some/key", fs.ErrNotExist),
			want: true,
		},
		{
			name: "gcs sentinel, wrapped",
			err:  fmt.Errorf("gcs: download '%s': %w", "some/key", gcs.ErrObjectNotExist),
			want: true,
		},
		{
			name: "s3 no such key, wrapped twice",
			err: fmt.Errorf("storage.ReadFile(%s): %w", "some/key",
				fmt.Errorf("s3: download '%s': %w", "some/key", minio.ErrorResponse{Code: minio.NoSuchKey})),
			want: true,
		},
		{
			name: "s3 head not found",
			err:  minio.ErrorResponse{Code: notFoundCode, StatusCode: http.StatusNotFound},
			want: true,
		},
		{
			name: "s3 no such multipart upload",
			err:  minio.ErrorResponse{Code: minio.NoSuchUpload, StatusCode: http.StatusNotFound},
			want: true,
		},
		{
			// minio labels a bodyless 404 as a missing bucket whenever the request carries no
			// object name, which is every listing
			name: "s3 no such bucket",
			err:  fmt.Errorf("s3: list '%s': %w", "some/dir/", minio.ErrorResponse{Code: minio.NoSuchBucket, StatusCode: http.StatusNotFound}),
			want: false,
		},
		{
			name: "s3 internal error",
			err:  minio.ErrorResponse{Code: "InternalError", StatusCode: http.StatusInternalServerError},
			want: false,
		},
		{
			name: "plain backend error",
			err:  errBackendDown,
			want: false,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			require.Equal(t, testCase.want, IsNotFound(testCase.err))
		})
	}
}
