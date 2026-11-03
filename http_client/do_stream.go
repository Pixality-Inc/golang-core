package http_client

import (
	"bytes"
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"time"

	http2 "github.com/pixality-inc/golang-core/http"
	"github.com/pixality-inc/golang-core/timetrack"
)

const defaultContentType = "application/json"

// StreamConnectTimeout bounds the dial and the tls handshake of DoStream. Timeout() of a stream
// client is the wait for the first answer of a slow server (minutes for a model), a host that does
// not accept the connection is known to be down much sooner
const StreamConnectTimeout = 30 * time.Second

// streamConnectTimeout is the connect deadline for a header timeout: StreamConnectTimeout, or the
// header timeout when it is shorter
func streamConnectTimeout(timeout time.Duration) time.Duration {
	if timeout > 0 && timeout < StreamConnectTimeout {
		return timeout
	}

	return StreamConnectTimeout
}

// newStreamClient builds the client behind DoStream. it is net/http and not fasthttp: fasthttp
// applies one read deadline to the headers and the whole body and can not abort a request in
// flight, while a stream needs a deadline for the headers only and cancellation by the context.
// the connection settings, tls and redirects follow the same config as the fasthttp client.
// the transport gets a copy of tlsConfig: net/http adds h2 to its NextProtos, the fasthttp client
// sharing it would then offer h2 over alpn and fail on the http/2 answer
func newStreamClient(config Config, tlsConfig *tls.Config, connectTimeout time.Duration) *http.Client {
	dialer := &net.Dialer{Timeout: connectTimeout}

	transport := &http.Transport{
		DialContext:           dialer.DialContext,
		TLSClientConfig:       tlsConfig.Clone(),
		TLSHandshakeTimeout:   connectTimeout,
		ResponseHeaderTimeout: config.Timeout(),
		ForceAttemptHTTP2:     true,
		MaxConnsPerHost:       config.MaxConnsPerHost(),
		MaxIdleConnsPerHost:   config.MaxConnsPerHost(),
		IdleConnTimeout:       config.MaxIdleConnDuration(),
		// the body goes to the caller as the server sent it, like the fasthttp client does
		DisableCompression: true,
	}

	client := &http.Client{Transport: transport}

	if !config.FollowRedirects() {
		client.CheckRedirect = func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		}
	}

	return client
}

// DoStream sends a request of any method and hands back the response body as a stream, for a
// body that is read while the server still writes it (server sent events, a proxied response).
//
// unlike GetStream the response is returned for every status code: the caller reads the status
// and the body of an error itself. Timeout() bounds waiting for the response headers, connecting
// and the tls handshake are bounded by StreamConnectTimeout (or Timeout() when shorter). the body
// has no deadline and ends with the context: canceling ctx aborts the request and closes the
// connection, also while the body is being read. the caller closes the body.
// a stream can not be replayed, so there are no retries and no circuit breaker
func (c *ClientImpl) DoStream(ctx context.Context, method, uri string, opts ...RequestOption) (StreamResponse, error) {
	requestTimeTracker := timetrack.New(ctx)

	cfg := applyOptions(opts...)
	url := c.makeUrl(uri)

	request, err := c.newStreamRequest(ctx, method, url, cfg)
	if err != nil {
		return nil, err
	}

	response, err := c.streamClient.Do(request) //nolint:bodyclose // the body is handed to the caller
	if err != nil {
		if ctx.Err() != nil {
			err = ctx.Err()
		}

		c.logStreamRequest(ctx, method, url, 0, nil, err, requestTimeTracker)

		return nil, err
	}

	headers := make(Headers, len(response.Header))
	for key, values := range response.Header {
		headers[key] = append([]string(nil), values...)
	}

	c.logStreamRequest(ctx, method, url, response.StatusCode, headers, nil, requestTimeTracker)

	return &StreamResponseImpl{
		StatusCode: response.StatusCode,
		Headers:    headers,
		Body:       response.Body,
	}, nil
}

func (c *ClientImpl) newStreamRequest(ctx context.Context, method, url string, cfg *RequestConfig) (*http.Request, error) {
	body, contentType, err := streamRequestBody(cfg)
	if err != nil {
		return nil, err
	}

	request, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	if c.config.UseRequestId() {
		if requestId, ok := ctx.Value(http2.RequestIdValueKey).(string); ok && requestId != "" {
			request.Header.Set("X-Request-Id", requestId)
		}
	}

	for headerKey, headerValues := range c.config.BaseHeaders() {
		for _, headerValue := range headerValues {
			request.Header.Add(headerKey, headerValue)
		}
	}

	for headerKey, headerValues := range cfg.Headers {
		for _, headerValue := range headerValues {
			request.Header.Add(headerKey, headerValue)
		}
	}

	if contentType != "" && request.Header.Get("Content-Type") == "" {
		request.Header.Set("Content-Type", contentType)
	}

	if len(cfg.QueryParams) > 0 {
		query := request.URL.Query()
		for key, value := range cfg.QueryParams {
			query.Add(key, value)
		}

		request.URL.RawQuery = query.Encode()
	}

	return request, nil
}

// streamRequestBody is the body of the request and its default content type, chosen the way
// applyRequestConfig chooses them for the fasthttp client
func streamRequestBody(cfg *RequestConfig) (io.Reader, string, error) {
	if cfg.FormData != nil && len(cfg.Body) > 0 {
		return nil, "", ErrBothFormDataAndBody
	}

	if cfg.FormData != nil {
		formData, ok := cfg.FormData.(*FormDataImpl)
		if !ok {
			return nil, "", ErrInvalidFormDataType
		}

		body, contentType, err := formData.build()
		if err != nil {
			return nil, "", err
		}

		return bytes.NewReader(body.Bytes()), contentType, nil
	}

	if len(cfg.Body) > 0 {
		return bytes.NewReader(cfg.Body), defaultContentType, nil
	}

	return http.NoBody, "", nil
}
