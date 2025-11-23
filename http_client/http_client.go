package http_client

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/valyala/fasthttp"

	http2 "github.com/pixality-inc/golang-core/http"
	"github.com/pixality-inc/golang-core/json"
	"github.com/pixality-inc/golang-core/logger"
	"github.com/pixality-inc/golang-core/timetrack"
)

var (
	ErrNotFound       = errors.New("not found")
	ErrBadRequest     = errors.New("bad request")
	ErrNon200HttpCode = errors.New("non-200 http status code")
)

type Client interface {
	Get(ctx context.Context, uri string, opts ...RequestOption) (*Response, error)
	Post(ctx context.Context, uri string, opts ...RequestOption) (*Response, error)
	Put(ctx context.Context, uri string, opts ...RequestOption) (*Response, error)
	Patch(ctx context.Context, uri string, opts ...RequestOption) (*Response, error)
	Delete(ctx context.Context, uri string, opts ...RequestOption) (*Response, error)
	Head(ctx context.Context, uri string, opts ...RequestOption) (*Response, error)
	Options(ctx context.Context, uri string, opts ...RequestOption) (*Response, error)

	Do(ctx context.Context, method, uri string, opts ...RequestOption) (*Response, error)
}

type ClientImpl struct {
	log    logger.Loggable
	config Config
	client *fasthttp.Client
}

func NewClientImpl(
	log logger.Loggable,
	config Config,
) *ClientImpl {
	readTimeout := config.ReadTimeout()
	if readTimeout == 0 {
		readTimeout = config.Timeout()
	}

	writeTimeout := config.WriteTimeout()
	if writeTimeout == 0 {
		writeTimeout = config.Timeout()
	}

	client := &fasthttp.Client{
		ReadTimeout:              readTimeout,
		WriteTimeout:             writeTimeout,
		MaxConnsPerHost:          config.MaxConnsPerHost(),
		MaxIdleConnDuration:      config.MaxIdleConnDuration(),
		MaxConnWaitTimeout:       config.MaxConnWaitTimeout(),
		NoDefaultUserAgentHeader: false,
		DisablePathNormalizing:   false,
	}

	if config.InsecureSkipVerify() {
		client.TLSConfig = &tls.Config{
			InsecureSkipVerify: true, // nolint:gosec
		}
	}

	return &ClientImpl{
		log:    log,
		config: config,
		client: client,
	}
}

func (c *ClientImpl) Get(ctx context.Context, uri string, opts ...RequestOption) (*Response, error) {
	return c.Do(ctx, http.MethodGet, uri, opts...)
}

func (c *ClientImpl) Post(ctx context.Context, uri string, opts ...RequestOption) (*Response, error) {
	return c.Do(ctx, http.MethodPost, uri, opts...)
}

func (c *ClientImpl) Put(ctx context.Context, uri string, opts ...RequestOption) (*Response, error) {
	return c.Do(ctx, http.MethodPut, uri, opts...)
}

func (c *ClientImpl) Patch(ctx context.Context, uri string, opts ...RequestOption) (*Response, error) {
	return c.Do(ctx, http.MethodPatch, uri, opts...)
}

func (c *ClientImpl) Delete(ctx context.Context, uri string, opts ...RequestOption) (*Response, error) {
	return c.Do(ctx, http.MethodDelete, uri, opts...)
}

func (c *ClientImpl) Head(ctx context.Context, uri string, opts ...RequestOption) (*Response, error) {
	return c.Do(ctx, http.MethodHead, uri, opts...)
}

func (c *ClientImpl) Options(ctx context.Context, uri string, opts ...RequestOption) (*Response, error) {
	return c.Do(ctx, http.MethodOptions, uri, opts...)
}

func (c *ClientImpl) Do(ctx context.Context, method, uri string, opts ...RequestOption) (*Response, error) {
	cfg := applyOptions(opts...)

	if c.config.RetryPolicy() != nil {
		return c.doWithRetry(ctx, func() (*Response, error) {
			return c.performRequest(ctx, method, uri, cfg)
		})
	}

	return c.performRequest(ctx, method, uri, cfg)
}

func (c *ClientImpl) makeUrl(uri string) string {
	baseUrl := c.config.BaseUrl()
	if baseUrl == "" {
		return uri
	}
	return baseUrl + uri
}

func AsJson[OUT any](response *Response, defaultValue OUT) (*TypedResponse[OUT], error) {
	typedResponse := &TypedResponse[OUT]{
		StatusCode: response.StatusCode,
		Headers:    response.Headers,
		Body:       response.Body,
		Entity:     defaultValue,
	}

	if err := json.Unmarshal(response.Body, &typedResponse.Entity); err != nil {
		return nil, err
	}

	return typedResponse, nil
}

func (c *ClientImpl) applyRequestConfig(ctx context.Context, req *fasthttp.Request, cfg *RequestConfig) error {
	if c.config.UseRequestId() {
		if requestId, ok := ctx.Value(http2.RequestIdValueKey).(string); ok && requestId != "" {
			req.Header.Set("X-Request-Id", requestId)
		}
	}

	for headerKey, headerValues := range c.config.BaseHeaders() {
		for _, headerValue := range headerValues {
			req.Header.Add(headerKey, headerValue)
		}
	}

	for headerKey, headerValues := range cfg.Headers {
		for _, headerValue := range headerValues {
			req.Header.Add(headerKey, headerValue)
		}
	}

	if len(cfg.QueryParams) > 0 {
		args := req.URI().QueryArgs()
		for key, value := range cfg.QueryParams {
			args.Add(key, value)
		}
	}

	if cfg.FormData != nil {
		formData, ok := cfg.FormData.(*FormData)
		if !ok {
			return errors.New("invalid form data type")
		}

		body, contentType, err := formData.build()
		if err != nil {
			return err
		}

		req.SetBody(body.Bytes())
		req.Header.SetContentType(contentType)
	} else if len(cfg.Body) > 0 {
		req.SetBody(cfg.Body)

		if len(req.Header.ContentType()) == 0 {
			req.Header.SetContentType("application/json")
		}
	}

	return nil
}

func (c *ClientImpl) performRequest(ctx context.Context, method, uri string, cfg *RequestConfig) (*Response, error) {
	requestTimeTracker := timetrack.New()

	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	url := c.makeUrl(uri)
	req.SetRequestURI(url)
	req.Header.SetMethod(method)

	if err := c.applyRequestConfig(ctx, req, cfg); err != nil {
		return nil, err
	}

	err := c.client.DoTimeout(req, resp, c.config.Timeout())

	if ctx.Err() != nil {
		return nil, ctx.Err()
	}

	response := &Response{
		StatusCode: resp.StatusCode(),
		Headers:    make(Headers),
		Body:       make([]byte, len(resp.Body())),
	}

	copy(response.Body, resp.Body())

	resp.Header.VisitAll(func(key, value []byte) {
		headerKey := string(key)
		headerValue := string(value)
		response.Headers[headerKey] = append(response.Headers[headerKey], headerValue)
	})

	c.logRequest(ctx, method, url, response, err, requestTimeTracker)

	if err != nil {
		return response, err
	}

	return c.handleStatusCode(response)
}

func (c *ClientImpl) logRequest(
	ctx context.Context,
	method, url string,
	response *Response,
	err error,
	tracker *timetrack.TimeTracker,
) {
	tracker.Finish()

	log := c.log.GetLogger(ctx)

	fields := map[string]any{
		"logger":         c.config.Name(),
		"method":         method,
		"url":            url,
		"success":        err == nil && (response == nil || (response.StatusCode >= 200 && response.StatusCode < 300)),
		"execution_time": tracker.Duration().Milliseconds(),
	}

	if response != nil {
		if value, ok := response.Headers["Content-Type"]; ok {
			fields["content_type"] = strings.Join(value, ",")
		}

		fields["body_bytes"] = len(response.Body)
		fields["status_code"] = response.StatusCode
	}

	logWithFields := log.WithFields(fields)

	if err != nil {
		logWithFields.WithError(err).Error(url)
	} else if response != nil && (response.StatusCode < 200 || response.StatusCode >= 300) {
		logWithFields.Warn(url)
	} else {
		logWithFields.Debug(url)
	}
}

func (c *ClientImpl) handleStatusCode(response *Response) (*Response, error) {
	switch {
	case response.StatusCode >= 200 && response.StatusCode <= 299:
		return response, nil

	case response.StatusCode == http.StatusNotFound:
		if len(response.Body) > 0 {
			return response, fmt.Errorf("%w: %s", ErrNotFound, response.Body)
		}
		return response, ErrNotFound

	case response.StatusCode == http.StatusBadRequest:
		if len(response.Body) > 0 {
			return response, fmt.Errorf("%w: %s", ErrBadRequest, response.Body)
		}
		return response, ErrBadRequest

	default:
		if len(response.Body) > 0 {
			return response, fmt.Errorf("%w: %d: %s", ErrNon200HttpCode, response.StatusCode, response.Body)
		}
		return response, fmt.Errorf("%w: %d", ErrNon200HttpCode, response.StatusCode)
	}
}
