package http_client

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/pixality-inc/golang-core/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	doStreamFirstChunk  = "data: first\n\n"
	doStreamSecondChunk = "data: second\n\n"
	doStreamWaitTimeout = 5 * time.Second
)

// writeTestChunk writes a piece of a test response; a client gone early is not an error here
func writeTestChunk(t *testing.T, w http.ResponseWriter, chunk string) {
	t.Helper()

	if _, err := w.Write([]byte(chunk)); err != nil {
		t.Logf("test response write: %v", err)
	}
}

func newDoStreamClient(t *testing.T, config Config) *ClientImpl {
	t.Helper()

	client, err := NewClientImpl(logger.NewLoggableImplWithService("test"), config)
	require.NoError(t, err)

	return client
}

func TestClientImpl_DoStream_PostsBodyAndStreamsChunks(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/v1/responses", r.URL.Path)
		assert.Equal(t, "1", r.URL.Query().Get("page"))
		assert.Equal(t, "Bearer upstream", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "base", r.Header.Get("X-Base"))

		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.JSONEq(t, `{"stream":true}`, string(body))

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)

		flusher, ok := w.(http.Flusher)
		assert.True(t, ok)

		writeTestChunk(t, w, doStreamFirstChunk)
		flusher.Flush()

		<-release

		writeTestChunk(t, w, doStreamSecondChunk)
	}))
	defer server.Close()

	config := newTestConfig(server.URL)
	config.baseHeaders = Headers{"X-Base": {"base"}}

	client := newDoStreamClient(t, config)

	response, err := client.DoStream(
		t.Context(),
		http.MethodPost,
		"/v1/responses",
		WithBody([]byte(`{"stream":true}`)),
		WithHeader("Authorization", "Bearer upstream"),
		WithQueryParam("page", "1"),
	)
	require.NoError(t, err)

	body := response.GetBody()

	defer func() {
		require.NoError(t, body.Close())
	}()

	assert.Equal(t, http.StatusOK, response.GetStatusCode())
	assert.Equal(t, []string{"text/event-stream"}, response.GetHeaders()["Content-Type"])

	// the first chunk is readable while the server still holds the second one
	first := make([]byte, len(doStreamFirstChunk))
	_, err = io.ReadFull(body, first)
	require.NoError(t, err)
	assert.Equal(t, doStreamFirstChunk, string(first))

	close(release)

	rest, err := io.ReadAll(body)
	require.NoError(t, err)
	assert.Equal(t, doStreamSecondChunk, string(rest))
}

func TestClientImpl_DoStream_ReturnsErrorStatusWithBody(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "3")
		w.WriteHeader(http.StatusTooManyRequests)

		writeTestChunk(t, w, `{"error":"slow down"}`)
	}))
	defer server.Close()

	client := newDoStreamClient(t, newTestConfig(server.URL))

	response, err := client.DoStream(t.Context(), http.MethodPost, "/limited")
	require.NoError(t, err)

	body := response.GetBody()

	defer func() {
		require.NoError(t, body.Close())
	}()

	assert.Equal(t, http.StatusTooManyRequests, response.GetStatusCode())
	assert.Equal(t, []string{"3"}, response.GetHeaders()["Retry-After"])

	content, err := io.ReadAll(body)
	require.NoError(t, err)
	assert.JSONEq(t, `{"error":"slow down"}`, string(content))
}

func TestClientImpl_DoStream_TimeoutBoundsHeadersOnly(t *testing.T) {
	t.Parallel()

	const headerTimeout = 200 * time.Millisecond

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/silent" {
			select {
			case <-r.Context().Done():
			case <-time.After(doStreamWaitTimeout):
			}

			return
		}

		flusher, ok := w.(http.Flusher)
		assert.True(t, ok)

		w.WriteHeader(http.StatusOK)
		flusher.Flush()

		// the body outlives the header timeout several times over
		for range 4 {
			time.Sleep(headerTimeout)

			writeTestChunk(t, w, doStreamFirstChunk)
			flusher.Flush()
		}
	}))
	defer server.Close()

	config := newTestConfig(server.URL)
	config.timeout = headerTimeout

	client := newDoStreamClient(t, config)

	_, err := client.DoStream(t.Context(), http.MethodPost, "/silent") //nolint:bodyclose // no response on error
	require.Error(t, err)

	response, err := client.DoStream(t.Context(), http.MethodPost, "/slow")
	require.NoError(t, err)

	body := response.GetBody()

	defer func() {
		require.NoError(t, body.Close())
	}()

	content, err := io.ReadAll(body)
	require.NoError(t, err)
	assert.Len(t, content, 4*len(doStreamFirstChunk))
}

func TestClientImpl_DoStream_ContextCancelAbortsTheBody(t *testing.T) {
	t.Parallel()

	serverDone := make(chan struct{})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer close(serverDone)

		flusher, ok := w.(http.Flusher)
		assert.True(t, ok)

		w.WriteHeader(http.StatusOK)

		writeTestChunk(t, w, doStreamFirstChunk)
		flusher.Flush()

		select {
		case <-r.Context().Done():
		case <-time.After(doStreamWaitTimeout):
		}
	}))
	defer server.Close()

	client := newDoStreamClient(t, newTestConfig(server.URL))

	ctx, cancel := context.WithCancel(t.Context())

	response, err := client.DoStream(ctx, http.MethodPost, "/endless")
	require.NoError(t, err)

	body := response.GetBody()

	defer func() {
		require.NoError(t, body.Close())
	}()

	first := make([]byte, len(doStreamFirstChunk))
	_, err = io.ReadFull(body, first)
	require.NoError(t, err)

	cancel()

	_, err = io.ReadAll(body)
	require.ErrorIs(t, err, context.Canceled)

	select {
	case <-serverDone:
	case <-time.After(doStreamWaitTimeout):
		t.Fatal("the server did not see the connection close")
	}
}

func TestClientImpl_DoStream_CanceledBeforeHeaders(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, r *http.Request) {
		<-r.Context().Done()
	}))
	defer server.Close()

	client := newDoStreamClient(t, newTestConfig(server.URL))

	ctx, cancel := context.WithCancel(t.Context())

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	_, err := client.DoStream(ctx, http.MethodPost, "/hang") //nolint:bodyclose // no response on error
	require.ErrorIs(t, err, context.Canceled)
}

func TestClientImpl_DoStream_RedirectsFollowConfig(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/old" {
			http.Redirect(w, r, "/new", http.StatusFound)

			return
		}

		writeTestChunk(t, w, "moved")
	}))
	defer server.Close()

	client := newDoStreamClient(t, newTestConfig(server.URL))

	response, err := client.DoStream(t.Context(), http.MethodGet, "/old")
	require.NoError(t, err)
	require.NoError(t, response.GetBody().Close())
	assert.Equal(t, http.StatusFound, response.GetStatusCode())

	following := newTestConfig(server.URL)
	following.followRedirects = true

	response, err = newDoStreamClient(t, following).DoStream(t.Context(), http.MethodGet, "/old")
	require.NoError(t, err)

	content, err := io.ReadAll(response.GetBody())
	require.NoError(t, err)
	require.NoError(t, response.GetBody().Close())
	assert.Equal(t, http.StatusOK, response.GetStatusCode())
	assert.Equal(t, "moved", string(content))
}

func TestClientImpl_DoStream_FormDataAndBodyConflict(t *testing.T) {
	t.Parallel()

	client := newDoStreamClient(t, newTestConfig("http://127.0.0.1:1"))

	formData := NewFormDataImpl()
	require.NoError(t, formData.AddField("model", "gpt"))

	_, err := client.DoStream( //nolint:bodyclose // no response on error
		t.Context(),
		http.MethodPost,
		"/edits",
		WithFormData(formData),
		WithBody([]byte("{}")),
	)
	require.ErrorIs(t, err, ErrBothFormDataAndBody)
}

func TestStreamConnectTimeout(t *testing.T) {
	t.Parallel()

	assert.Equal(t, StreamConnectTimeout, streamConnectTimeout(3*time.Minute), "a long header timeout")
	assert.Equal(t, StreamConnectTimeout, streamConnectTimeout(0), "no header timeout")
	assert.Equal(t, 5*time.Second, streamConnectTimeout(5*time.Second), "a header timeout shorter than the default")
}

func TestClientImpl_DoStream_ConnectTimeoutBoundsTheHandshake(t *testing.T) {
	t.Parallel()

	const connectTimeout = 200 * time.Millisecond

	// a tcp server that accepts and never answers the tls handshake
	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)

	defer func() {
		require.NoError(t, listener.Close())
	}()

	testDone := make(chan struct{})
	defer close(testDone)

	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}

		<-testDone

		_ = conn.Close() //nolint:errcheck // the test is over
	}()

	config := newTestConfig("https://" + listener.Addr().String())
	config.timeout = time.Minute

	client := newDoStreamClient(t, config)
	client.streamClient = newStreamClient(config, &tls.Config{MinVersion: tls.VersionTLS12}, connectTimeout)

	startedAt := time.Now()

	_, err = client.DoStream(t.Context(), http.MethodPost, "/v1/responses") //nolint:bodyclose // no response on error
	require.Error(t, err)

	assert.Less(t, time.Since(startedAt), doStreamWaitTimeout, "the handshake is cut by the connect timeout, not the header one")
}

func TestClientImpl_DoStream_KeepsTheTLSConfigOfTheFasthttpClient(t *testing.T) {
	t.Parallel()

	newProtoServer := func() *httptest.Server {
		server := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			writeTestChunk(t, w, r.Proto)
		}))
		server.EnableHTTP2 = true
		server.StartTLS()

		return server
	}

	streamServer := newProtoServer()
	defer streamServer.Close()

	server := newProtoServer()
	defer server.Close()

	config := newTestConfig("")
	config.insecureSkipVerify = true

	client := newDoStreamClient(t, config)

	response, err := client.DoStream(t.Context(), http.MethodGet, streamServer.URL)
	require.NoError(t, err)

	body, err := io.ReadAll(response.GetBody())
	require.NoError(t, err)
	require.NoError(t, response.GetBody().Close())
	assert.Equal(t, "HTTP/2.0", string(body))

	// net/http adds h2 to NextProtos of the tls config it gets, a shared one makes fasthttp offer
	// h2 to a host it connects to afterwards and fail on the http/2 answer
	plainResponse, err := client.Get(t.Context(), server.URL)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, plainResponse.GetStatusCode())
	assert.Equal(t, "HTTP/1.1", string(plainResponse.GetBody()))
}
