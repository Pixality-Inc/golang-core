package http

import (
	"bufio"
	"errors"

	"github.com/valyala/fasthttp"
)

var ErrNoStreamWriter = errors.New("no stream writer")

// StreamWriter writes the body of a streamed response. fasthttp runs it on a goroutine of its own
// from the moment it is handed over, and sends its output once the handler chain has returned with
// the status and the headers. each Flush sends what was written so far to the client, and a failed
// Write or Flush means the client is gone. fasthttp may release the request while the writer still
// runs, so the writer does not use the request context
type StreamWriter = func(writer *bufio.Writer)

// StreamResponse is the result of a handler whose body is produced while it is sent: server sent
// events, a proxied upstream response. the body is not buffered, so the request logger reports its
// size as StreamedResponseBytes
type StreamResponse interface {
	StatusCode() int
	Headers() map[string]string
	Writer() StreamWriter
}

type StreamResponseImpl struct {
	statusCode int
	headers    map[string]string
	writer     StreamWriter
}

// NewStreamResponse builds a streamed response; a zero status code means 200
func NewStreamResponse(statusCode int, headers map[string]string, writer StreamWriter) *StreamResponseImpl {
	if statusCode == 0 {
		statusCode = fasthttp.StatusOK
	}

	return &StreamResponseImpl{
		statusCode: statusCode,
		headers:    headers,
		writer:     writer,
	}
}

func (r *StreamResponseImpl) StatusCode() int {
	return r.statusCode
}

func (r *StreamResponseImpl) Headers() map[string]string {
	return r.headers
}

// Writer of a nil response is nil, so HandleStream renders a typed nil response as ErrNoStreamWriter
func (r *StreamResponseImpl) Writer() StreamWriter {
	if r == nil {
		return nil
	}

	return r.writer
}

// HandleStream sets the status and the headers of the response and hands its writer to fasthttp
func HandleStream(ctx *fasthttp.RequestCtx, response StreamResponse) {
	if response == nil || response.Writer() == nil {
		InternalServerError(ctx, ErrNoStreamWriter)

		return
	}

	ctx.SetStatusCode(response.StatusCode())

	for key, value := range response.Headers() {
		ctx.Response.Header.Set(key, value)
	}

	ctx.SetBodyStreamWriter(response.Writer())
}
