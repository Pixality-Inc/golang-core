package storage

import (
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"os"
	"syscall"
	"testing"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/googleapi"
)

func TestIsRejected(t *testing.T) {
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
			name: "os provider permission, wrapped",
			err: fmt.Errorf("storage.WriteFile(%s): %w", "some/key",
				&fs.PathError{Op: "open", Path: "/data/some/key", Err: syscall.EACCES}),
			want: true,
		},
		{
			// a dropped connection carries the same errno as a file the process may not open
			name: "socket not permitted",
			err: fmt.Errorf("storage.ReadFile(%s): %w", "some/key", &url.Error{
				Op:  "Get",
				URL: "https://s3.example.com/bucket/some/key",
				Err: &net.OpError{
					Op:  "dial",
					Net: "tcp",
					Err: os.NewSyscallError("connect", syscall.EPERM),
				},
			}),
			want: false,
		},
		{
			name: "s3 access denied, wrapped twice",
			err: fmt.Errorf("storage.ReadFile(%s): %w", "some/key",
				fmt.Errorf("s3: download '%s': %w", "some/key", minio.ErrorResponse{Code: "AccessDenied", StatusCode: http.StatusForbidden})),
			want: true,
		},
		{
			name: "s3 unauthorized",
			err:  minio.ErrorResponse{Code: "Unauthorized", StatusCode: http.StatusUnauthorized},
			want: true,
		},
		{
			name: "s3 precondition failed",
			err: fmt.Errorf("s3: download '%s': %w", "some/key.json",
				minio.ErrorResponse{Code: "PreconditionFailed", StatusCode: http.StatusPreconditionFailed}),
			want: true,
		},
		{
			// a node with a drifting clock gets it, a later attempt does not
			name: "s3 request time too skewed",
			err:  minio.ErrorResponse{Code: minio.RequestTimeTooSkewed, StatusCode: http.StatusForbidden},
			want: false,
		},
		{
			name: "gcs forbidden, wrapped",
			err:  fmt.Errorf("gcs: upload '%s': %w", "some/key", &googleapi.Error{Code: http.StatusForbidden}),
			want: true,
		},
		{
			name: "gcs unauthorized",
			err:  &googleapi.Error{Code: http.StatusUnauthorized},
			want: true,
		},
		{
			name: "gcs precondition failed",
			err:  &googleapi.Error{Code: http.StatusPreconditionFailed},
			want: true,
		},
		{
			// S3 answers a client too slow to send its body with 400, which is worth waiting out
			name: "s3 request timeout",
			err:  minio.ErrorResponse{Code: "RequestTimeout", StatusCode: http.StatusBadRequest},
			want: false,
		},
		{
			name: "s3 slow down",
			err:  minio.ErrorResponse{Code: "SlowDown", StatusCode: http.StatusTooManyRequests},
			want: false,
		},
		{
			name: "s3 service unavailable",
			err:  minio.ErrorResponse{Code: "ServiceUnavailable", StatusCode: http.StatusServiceUnavailable},
			want: false,
		},
		{
			name: "s3 no such key",
			err:  minio.ErrorResponse{Code: minio.NoSuchKey, StatusCode: http.StatusNotFound},
			want: false,
		},
		{
			name: "gcs service unavailable",
			err:  &googleapi.Error{Code: http.StatusServiceUnavailable},
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

			require.Equal(t, testCase.want, IsRejected(testCase.err))
		})
	}
}
