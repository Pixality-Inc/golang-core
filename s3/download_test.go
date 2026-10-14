package s3

import (
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	r2FakeBody = `{"plain":"json object stored without content encoding"}`
	r2FakeEtag = `"152b6f91affad3cddd5c18a33a7f6def-1"`
)

// newR2LikeServer mimics cloudflare r2 serving a plain json object: asked for gzip
// it compresses the body on the fly and weakens the etag, and it compares If-Match
// strongly so the weak etag never matches. it records the Accept-Encoding of every
// request.
func newR2LikeServer(t *testing.T) (*httptest.Server, func() []string) {
	t.Helper()

	var (
		mutex     sync.Mutex
		encodings []string
	)

	var compressed bytes.Buffer

	gz := gzip.NewWriter(&compressed)
	_, err := gz.Write([]byte(r2FakeBody))
	require.NoError(t, err)
	require.NoError(t, gz.Close())

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		acceptEncoding := r.Header.Get("Accept-Encoding")

		mutex.Lock()

		encodings = append(encodings, acceptEncoding)

		mutex.Unlock()

		if ifMatch := r.Header.Get("If-Match"); ifMatch != "" && ifMatch != r2FakeEtag {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusPreconditionFailed)

			if _, err := io.WriteString(w, `<Error><Code>PreconditionFailed</Code><Message>At least one of the pre-conditions you specified did not hold.</Message></Error>`); err != nil {
				t.Errorf("write error body: %v", err)
			}

			return
		}

		body := []byte(r2FakeBody)
		etag := r2FakeEtag

		if strings.Contains(acceptEncoding, "gzip") {
			body = compressed.Bytes()
			etag = "W/" + r2FakeEtag

			w.Header().Set("Content-Encoding", "gzip")
		}

		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Content-Length", strconv.Itoa(len(body)))
		w.Header().Set("ETag", etag)
		w.Header().Set("Last-Modified", "Wed, 14 Oct 2026 13:45:53 GMT")
		w.WriteHeader(http.StatusOK)

		if r.Method != http.MethodHead {
			if _, err := w.Write(body); err != nil {
				t.Errorf("write body: %v", err)
			}
		}
	}))
	t.Cleanup(server.Close)

	return server, func() []string {
		mutex.Lock()
		defer mutex.Unlock()

		return append([]string(nil), encodings...)
	}
}

func downloadAll(ctx context.Context, client Client, objectName string) ([]byte, error) {
	reader, err := client.Download(ctx, objectName)
	if err != nil {
		return nil, err
	}

	defer func() { _ = reader.Close() }()

	return io.ReadAll(reader)
}

func TestDownload_GzipAcceptEncodingAgainstR2(t *testing.T) {
	t.Parallel()

	t.Run("forced gzip hits the weak etag precondition", func(t *testing.T) {
		t.Parallel()

		server, encodings := newR2LikeServer(t)

		client := NewClient("r2-test", server.URL, "us-east-1", "", "", "bucket", "base-dir", "", true)
		t.Cleanup(client.Close)

		_, err := downloadAll(context.Background(), client, "id/id.json")
		require.ErrorContains(t, err, "pre-conditions")

		for _, encoding := range encodings() {
			require.Equal(t, "gzip", encoding, "default must keep advertising gzip")
		}
	})

	t.Run("opt out reads the stored bytes", func(t *testing.T) {
		t.Parallel()

		server, encodings := newR2LikeServer(t)

		client := NewClient(
			"r2-test", server.URL, "us-east-1", "", "", "bucket", "base-dir", "", true,
			WithForceGzipAcceptEncoding(false),
		)
		t.Cleanup(client.Close)

		data, err := downloadAll(context.Background(), client, "id/id.json")
		require.NoError(t, err)
		require.JSONEq(t, r2FakeBody, string(data))

		require.NotEmpty(t, encodings())

		for _, encoding := range encodings() {
			require.NotContains(t, encoding, "gzip", "opted out client must not advertise gzip")
		}
	})
}
