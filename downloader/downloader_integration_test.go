package downloader_test

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/pixality-inc/golang-core/downloader"
	"github.com/pixality-inc/golang-core/http_client"
)

func TestDownloader_Download_Integration(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)

		_, err := w.Write([]byte("hello"))
		if err != nil {
			t.Fatal(err)
		}
	}))
	defer server.Close()

	cfg := &http_client.ConfigYaml{TimeoutValue: 5 * time.Second}

	testDownloader, err := downloader.NewDownloader(cfg)
	require.NoError(t, err)

	body, err := testDownloader.Download(t.Context(), server.URL)

	require.NoError(t, err)
	require.Equal(t, []byte("hello"), body)
}
