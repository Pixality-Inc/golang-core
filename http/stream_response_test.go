package http

import (
	"bufio"
	"context"
	"io"
	"net"
	nethttp "net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

const (
	streamFirstChunk  = "data: first\n\n"
	streamSecondChunk = "data: second\n\n"
	streamWaitTimeout = 5 * time.Second
)

// serveStream runs handler behind the request logger on a local listener and returns its url
func serveStream(t *testing.T, handler fasthttp.RequestHandler) string {
	t.Helper()

	var listenConfig net.ListenConfig

	listener, err := listenConfig.Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)

	server := &fasthttp.Server{Handler: RequestLogHandler(handler)}

	go func() {
		_ = server.Serve(listener) //nolint:errcheck // the listener is closed by the cleanup
	}()

	t.Cleanup(func() {
		require.NoError(t, server.Shutdown())
	})

	return "http://" + listener.Addr().String()
}

func TestHandleStream_SendsStatusHeadersAndFlushedChunks(t *testing.T) {
	t.Parallel()

	release := make(chan struct{})

	url := serveStream(t, func(ctx *fasthttp.RequestCtx) {
		HandleStream(ctx, NewStreamResponse(
			fasthttp.StatusTooManyRequests,
			map[string]string{"Content-Type": "text/event-stream", "Retry-After": "7"},
			func(writer *bufio.Writer) {
				_, _ = writer.WriteString(streamFirstChunk) //nolint:errcheck // a failed write fails the flush
				_ = writer.Flush()                          //nolint:errcheck // the client reads the chunk or the test fails

				<-release

				_, _ = writer.WriteString(streamSecondChunk) //nolint:errcheck // a failed write fails the flush
				_ = writer.Flush()                           //nolint:errcheck // the client reads the chunk or the test fails
			},
		))
	})

	// the headers and the first chunk arrive while the writer still waits: neither the server nor
	// the request logger buffers the stream
	type firstRead struct {
		response *nethttp.Response
		chunk    string
		err      error
	}

	firstReads := make(chan firstRead, 1)

	go func() {
		request, err := nethttp.NewRequestWithContext(t.Context(), nethttp.MethodGet, url, nil)
		if err != nil {
			firstReads <- firstRead{err: err}

			return
		}

		response, err := nethttp.DefaultClient.Do(request) //nolint:bodyclose // closed by the test below
		if err != nil {
			firstReads <- firstRead{err: err}

			return
		}

		chunk := make([]byte, len(streamFirstChunk))
		_, err = io.ReadFull(response.Body, chunk)
		firstReads <- firstRead{response: response, chunk: string(chunk), err: err}
	}()

	var first firstRead

	select {
	case first = <-firstReads:
	case <-time.After(streamWaitTimeout):
		close(release)
		t.Fatal("the first chunk did not arrive before the stream ended")
	}

	require.NoError(t, first.err)

	defer func() {
		require.NoError(t, first.response.Body.Close())
	}()

	assert.Equal(t, nethttp.StatusTooManyRequests, first.response.StatusCode)
	assert.Equal(t, "text/event-stream", first.response.Header.Get("Content-Type"))
	assert.Equal(t, "7", first.response.Header.Get("Retry-After"))
	assert.Equal(t, streamFirstChunk, first.chunk)

	close(release)

	rest, err := io.ReadAll(first.response.Body)
	require.NoError(t, err)
	assert.Equal(t, streamSecondChunk, string(rest))
}

func TestHandleStream_ZeroStatusMeansOk(t *testing.T) {
	t.Parallel()

	response := NewStreamResponse(0, nil, func(*bufio.Writer) {})

	assert.Equal(t, fasthttp.StatusOK, response.StatusCode())
}

func TestHandleStream_NoWriterIsInternalError(t *testing.T) {
	t.Parallel()

	renderer := NewResponseRenderer(&testProtoRenderer{})
	ctx := newRenderCtx(mediaTypeJSON)
	ctx.SetUserValue(ResponseRendererValueKey, renderer)

	HandleStream(ctx, NewStreamResponse(fasthttp.StatusOK, nil, nil))

	assert.Equal(t, fasthttp.StatusInternalServerError, ctx.Response.StatusCode())
	assert.False(t, ctx.Response.IsBodyStream())
}

func TestHandleStream_TypedNilResponseIsInternalError(t *testing.T) {
	t.Parallel()

	renderer := NewResponseRenderer(&testProtoRenderer{})
	ctx := newRenderCtx(mediaTypeJSON)
	ctx.SetUserValue(ResponseRendererValueKey, renderer)

	var response *StreamResponseImpl

	HandleStream(ctx, response)

	assert.Equal(t, fasthttp.StatusInternalServerError, ctx.Response.StatusCode())
	assert.False(t, ctx.Response.IsBodyStream())
}

func TestResponseBytes(t *testing.T) {
	t.Parallel()

	buffered := &fasthttp.RequestCtx{}
	buffered.SetBodyString("12345")

	assert.Equal(t, len("12345"), responseBytes(buffered))

	sized := &fasthttp.RequestCtx{}
	sized.SetBodyStream(strings.NewReader("12345"), len("12345"))

	assert.Equal(t, len("12345"), responseBytes(sized), "a stream of a known length")

	unsized := &fasthttp.RequestCtx{}
	// fasthttp takes a negative size for a body of an unknown length
	unsized.SetBodyStream(strings.NewReader("12345"), StreamedResponseBytes)

	assert.Equal(t, StreamedResponseBytes, responseBytes(unsized), "a stream of an unknown length")

	release := make(chan struct{})
	defer close(release)

	streamed := &fasthttp.RequestCtx{}
	streamed.SetBodyStreamWriter(func(writer *bufio.Writer) {
		_, _ = writer.WriteString(streamFirstChunk) //nolint:errcheck // nobody reads the stream here

		<-release
	})

	// reading the size of the stream would block until the writer returns
	measured := make(chan int, 1)

	go func() {
		measured <- responseBytes(streamed)
	}()

	select {
	case size := <-measured:
		assert.Equal(t, StreamedResponseBytes, size)
	case <-time.After(streamWaitTimeout):
		t.Fatal("the size of a streamed body must not drain the stream")
	}
}

func TestHandleStream_ClientGoneFailsTheWrite(t *testing.T) {
	t.Parallel()

	writeFailed := make(chan error, 1)
	firstSent := make(chan struct{})

	url := serveStream(t, func(ctx *fasthttp.RequestCtx) {
		HandleStream(ctx, NewStreamResponse(fasthttp.StatusOK, nil, func(writer *bufio.Writer) {
			_, _ = writer.WriteString(streamFirstChunk) //nolint:errcheck // a failed write fails the flush
			_ = writer.Flush()                          //nolint:errcheck // the client reads the chunk or the test fails

			close(firstSent)

			ticker := time.NewTicker(time.Millisecond)
			defer ticker.Stop()

			deadline := time.After(streamWaitTimeout)

			for {
				select {
				case <-ticker.C:
					_, _ = writer.WriteString(streamSecondChunk) //nolint:errcheck // a failed write fails the flush

					if err := writer.Flush(); err != nil {
						writeFailed <- err

						return
					}
				case <-deadline:
					writeFailed <- nil

					return
				}
			}
		}))
	})

	requestCtx, cancel := context.WithCancel(t.Context())

	request, err := nethttp.NewRequestWithContext(requestCtx, nethttp.MethodGet, url, nil)
	require.NoError(t, err)

	response, err := nethttp.DefaultClient.Do(request)
	require.NoError(t, err)

	<-firstSent

	cancel()
	require.NoError(t, response.Body.Close())

	select {
	case err := <-writeFailed:
		require.Error(t, err)
	case <-time.After(streamWaitTimeout):
		t.Fatal("the writer did not notice the client was gone")
	}
}
