package gcs

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"strings"
	"testing"
	"time"

	gstorage "cloud.google.com/go/storage"
	"github.com/pixality-inc/golang-core/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/api/googleapi"
	"google.golang.org/api/option"
)

func TestImpl_Stat(t *testing.T) {
	t.Parallel()

	modTime := time.Date(2026, time.September, 1, 12, 34, 56, 0, time.UTC)

	cases := []struct {
		name        string
		baseDir     string
		objectName  string
		fullName    string
		escapedPath string
		wantName    string
		wantSize    int64
	}{
		{
			name:        "nested object with base directory",
			baseDir:     "base",
			objectName:  "nested/report.json",
			fullName:    "base/nested/report.json",
			escapedPath: "/storage/v1/b/stat-bucket/o/base%2Fnested%2Freport.json",
			wantName:    "report.json",
			wantSize:    1024,
		},
		{
			name:        "empty object without base directory",
			objectName:  "empty.bin",
			fullName:    "empty.bin",
			escapedPath: "/storage/v1/b/stat-bucket/o/empty.bin",
			wantName:    "empty.bin",
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			requests := 0
			client := newStatTestClient(t, testCase.baseDir, func(req *http.Request) (*http.Response, error) {
				requests++

				assert.Equal(t, http.MethodGet, req.Method)
				assert.Equal(t, testCase.escapedPath, req.URL.EscapedPath())
				assert.Equal(t, "json", req.URL.Query().Get("alt"))
				assert.Equal(t, "full", req.URL.Query().Get("projection"))
				assert.Empty(t, req.URL.Query().Get("prefix"))
				assert.Empty(t, req.URL.Query().Get("delimiter"))

				body := fmt.Sprintf(`{"bucket":"stat-bucket","name":%q,"size":"%d","updated":%q}`, testCase.fullName, testCase.wantSize, modTime.Format(time.RFC3339))

				return statTestResponse(req, http.StatusOK, body), nil
			})

			info, err := NewStorageProvider(client).Stat(context.Background(), testCase.objectName)
			require.NoError(t, err)
			require.NotNil(t, info)
			assert.Equal(t, 1, requests, "Stat should issue only one object metadata request")
			assert.Equal(t, testCase.wantName, info.Name())
			assert.Equal(t, testCase.wantSize, info.Size())
			assert.Equal(t, modTime, info.ModTime())
			assert.Equal(t, fs.FileMode(0), info.Mode())
			assert.False(t, info.IsDir())
			assert.Nil(t, info.Sys())
		})
	}
}

func TestImpl_StatErrors(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		status     int
		wantAbsent bool
	}{
		{name: "missing exact object", status: http.StatusNotFound, wantAbsent: true},
		{name: "permission denied", status: http.StatusForbidden},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			requests := 0
			client := newStatTestClient(t, "base", func(req *http.Request) (*http.Response, error) {
				requests++

				assert.Equal(t, "/storage/v1/b/stat-bucket/o/base%2Fvirtual-directory", req.URL.EscapedPath())

				return statTestResponse(req, testCase.status, fmt.Sprintf(`{"error":{"code":%d,"message":"test error"}}`, testCase.status)), nil
			})

			info, err := client.Stat(context.Background(), "virtual-directory")
			require.Error(t, err)
			assert.Nil(t, info)
			assert.Equal(t, 1, requests, "Stat must not list a missing object as a virtual directory")
			assert.Contains(t, err.Error(), "gcs: stat 'base/virtual-directory'")
			assert.Equal(t, testCase.wantAbsent, storage.IsNotFound(err))

			if testCase.wantAbsent {
				require.ErrorIs(t, err, gstorage.ErrObjectNotExist)
			} else {
				var apiError *googleapi.Error
				require.ErrorAs(t, err, &apiError)
				assert.Equal(t, http.StatusForbidden, apiError.Code)
			}
		})
	}
}

func TestImpl_StatCanceledContext(t *testing.T) {
	t.Parallel()

	client := newStatTestClient(t, "", func(req *http.Request) (*http.Response, error) {
		return nil, req.Context().Err()
	})
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	info, err := client.Stat(ctx, "object.bin")
	require.ErrorIs(t, err, context.Canceled)
	assert.Nil(t, info)
	assert.False(t, storage.IsNotFound(err))
}

type statTestTransport func(*http.Request) (*http.Response, error)

func (transport statTestTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return transport(req)
}

func newStatTestClient(t *testing.T, baseDir string, transport statTestTransport) *Impl {
	t.Helper()

	raw, err := gstorage.NewClient(context.Background(),
		option.WithEndpoint("https://storage.test/storage/v1/"),
		option.WithHTTPClient(&http.Client{Transport: transport}),
		option.WithoutAuthentication(),
	)
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, raw.Close()) })

	return &Impl{client: raw, bucketName: "stat-bucket", baseDir: baseDir}
}

func statTestResponse(req *http.Request, status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": {"application/json"}},
		Body:       io.NopCloser(strings.NewReader(body)),
		Request:    req,
	}
}
