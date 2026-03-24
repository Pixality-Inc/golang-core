package http_client

import (
	"io"
	"sync"

	"github.com/valyala/fasthttp"
)

//go:generate mockgen -destination mocks/stream_response_gen.go -source stream_response.go
type StreamResponse interface {
	GetStatusCode() int
	GetHeaders() Headers
	GetBody() io.ReadCloser
}

type StreamResponseImpl struct {
	StatusCode int
	Headers    Headers
	Body       io.ReadCloser
}

func (r *StreamResponseImpl) GetStatusCode() int     { return r.StatusCode }
func (r *StreamResponseImpl) GetHeaders() Headers    { return r.Headers }
func (r *StreamResponseImpl) GetBody() io.ReadCloser { return r.Body }

type fasthttpStreamCloser struct {
	reader io.Reader
	req    *fasthttp.Request
	resp   *fasthttp.Response
	once   sync.Once
}

func (s *fasthttpStreamCloser) Read(p []byte) (int, error) {
	return s.reader.Read(p)
}

func (s *fasthttpStreamCloser) Close() error {
	s.once.Do(func() {
		fasthttp.ReleaseRequest(s.req)
		fasthttp.ReleaseResponse(s.resp)
	})
	return nil
}
