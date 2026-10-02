package s3

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/stretchr/testify/require"

	"github.com/pixality-inc/golang-core/storage"
)

func TestStat_HeadMetadata(t *testing.T) {
	t.Parallel()

	modTime := time.Date(2026, time.January, 2, 3, 4, 5, 0, time.UTC)
	requests := make(chan *http.Request, 2)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- r

		w.Header().Set("Content-Length", "1234")
		w.Header().Set("Last-Modified", modTime.Format(http.TimeFormat))
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	client := NewClient("stat-test", server.URL, "us-east-1", "", "", "bucket", "base-dir", "", true)
	t.Cleanup(client.Close)

	info, err := NewStorageProvider(client).Stat(context.Background(), "nested/file name.bin")
	require.NoError(t, err)
	require.Equal(t, "file name.bin", info.Name())
	require.Equal(t, int64(1234), info.Size())
	require.Equal(t, modTime, info.ModTime())
	require.False(t, info.IsDir())
	require.Zero(t, info.Mode())

	require.Len(t, requests, 1, "metadata retrieval must not download or list objects")
	request := <-requests
	require.Equal(t, http.MethodHead, request.Method)
	require.Equal(t, "/bucket/base-dir/nested/file name.bin", request.URL.Path)
	require.Empty(t, request.URL.RawQuery)
}

func TestStat_Errors(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name       string
		status     int
		wantCode   string
		wantAbsent bool
	}{
		{name: "missing object", status: http.StatusNotFound, wantCode: minio.NoSuchKey, wantAbsent: true},
		{name: "access denied", status: http.StatusForbidden, wantCode: minio.AccessDenied},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			requests := make(chan *http.Request, 2)
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requests <- r

				w.WriteHeader(testCase.status)
			}))
			t.Cleanup(server.Close)

			client := NewClient("stat-test", server.URL, "us-east-1", "", "", "bucket", "base-dir", "", true)
			t.Cleanup(client.Close)

			info, err := client.Stat(context.Background(), "nested/missing.bin")
			require.Error(t, err)
			require.Nil(t, info)
			require.ErrorContains(t, err, "s3: stat 'base-dir/nested/missing.bin'")
			require.Equal(t, testCase.wantAbsent, storage.IsNotFound(err))

			var response minio.ErrorResponse
			require.ErrorAs(t, err, &response)
			require.Equal(t, testCase.wantCode, response.Code)
			require.Equal(t, testCase.status, response.StatusCode)
			require.Len(t, requests, 1, "Stat must preserve exact object lookup on failure")
			request := <-requests
			require.Equal(t, http.MethodHead, request.Method)
			require.Equal(t, "/bucket/base-dir/nested/missing.bin", request.URL.Path)
			require.Empty(t, request.URL.RawQuery)
		})
	}
}

func TestStat_CanceledContext(t *testing.T) {
	t.Parallel()

	var calls atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		calls.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(server.Close)

	client := NewClient("stat-test", server.URL, "us-east-1", "", "", "bucket", "", "", true)
	t.Cleanup(client.Close)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	info, err := client.Stat(ctx, "file.bin")
	require.ErrorIs(t, err, context.Canceled)
	require.Nil(t, info)
	require.Zero(t, calls.Load())
}
