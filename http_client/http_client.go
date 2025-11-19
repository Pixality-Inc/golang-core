package http_client

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"strings"

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
	Do(req *http.Request) (*http.Response, error)
	Get(ctx context.Context, uri string, request *Request) (*Response, error)
	Post(ctx context.Context, uri string, body *bytes.Buffer, request *Request) (*Response, error)
	PostMultipart(ctx context.Context, uri string, formData *FormData, request *Request) (*Response, error)
}

type ClientImpl struct {
	log    logger.Loggable
	config Config
	client *http.Client
}

func NewClientImpl(
	log logger.Loggable,
	config Config,
) *ClientImpl {
	var transport *http.Transport

	if config.InsecureSkipVerify() {
		transport = &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: config.InsecureSkipVerify(), // nolint:gosec
			},
		}
	} else {
		defaultTransport, ok := http.DefaultTransport.(*http.Transport)
		if !ok {
			defaultTransport = &http.Transport{}
		}

		transport = defaultTransport.Clone()
	}

	return &ClientImpl{
		log:    log,
		config: config,
		client: &http.Client{
			Timeout:   config.Timeout(),
			Transport: transport,
		},
	}
}

func (c *ClientImpl) Do(req *http.Request) (*http.Response, error) {
	return c.client.Do(req)
}

func (c *ClientImpl) Get(ctx context.Context, uri string, request *Request) (*Response, error) {
	url := c.makeUrl(uri)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, bytes.NewBuffer(nil))
	if err != nil {
		return nil, err
	}

	c.applyRequest(ctx, req, request)

	return c.do(req)
}

func (c *ClientImpl) Post(ctx context.Context, uri string, body *bytes.Buffer, request *Request) (*Response, error) {
	url := c.makeUrl(uri)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, body)
	if err != nil {
		return nil, err
	}

	c.applyRequest(ctx, req, request)

	return c.do(req)
}

func (c *ClientImpl) PostMultipart(ctx context.Context, uri string, formData *FormData, request *Request) (*Response, error) {
	if err := formData.Close(); err != nil {
		return nil, err
	}

	url := c.makeUrl(uri)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, formData.Body())
	if err != nil {
		return nil, err
	}

	c.applyRequest(ctx, req, request)

	req.Header.Set("Content-Type", formData.ContentType())

	return c.do(req)
}

func (c *ClientImpl) makeUrl(uri string) string {
	return c.config.BaseUrl() + uri
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

func (c *ClientImpl) applyRequest(ctx context.Context, httpRequest *http.Request, request *Request) {
	if c.config.UseRequestId() {
		if requestId, ok := ctx.Value(http2.RequestIdValueKey).(string); ok && requestId != "" {
			httpRequest.Header.Set("X-Request-Id", requestId)
		}
	}

	for headerKey, headerValues := range c.config.BaseHeaders() {
		for _, headerValue := range headerValues {
			httpRequest.Header.Add(headerKey, headerValue)
		}
	}

	if request != nil {
		c.addQueryParams(httpRequest, request.QueryParams)
		c.addHeaders(httpRequest, request.Headers)
	}
}

func (c *ClientImpl) addQueryParams(req *http.Request, queryParams QueryParams) {
	query := req.URL.Query()

	for k, v := range queryParams {
		query.Add(k, v)
	}

	req.URL.RawQuery = query.Encode()
}

func (c *ClientImpl) addHeaders(req *http.Request, headers Headers) {
	for headerKey, headerValues := range headers {
		for _, headerValue := range headerValues {
			req.Header.Add(headerKey, headerValue)
		}
	}
}

func (c *ClientImpl) performRequest(ctx context.Context, req *http.Request) (*Response, error) {
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err = resp.Body.Close(); err != nil {
			c.log.GetLogger(ctx).WithError(err).Error("failed to close response body")
		}
	}()

	headers := make(Headers)

	maps.Copy(headers, resp.Header)

	response := &Response{
		StatusCode: resp.StatusCode,
		Headers:    headers,
		Body:       nil,
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return response, err
	}

	response.Body = respBody

	switch {
	case resp.StatusCode >= 200 && resp.StatusCode <= 299:
		return response, nil

	case resp.StatusCode == http.StatusNotFound:
		if len(response.Body) > 0 {
			return response, fmt.Errorf("%w: %s", ErrNotFound, respBody)
		} else {
			return response, ErrNotFound
		}

	case resp.StatusCode == http.StatusBadRequest:
		if len(response.Body) > 0 {
			return response, fmt.Errorf("%w: %s", ErrBadRequest, respBody)
		} else {
			return response, ErrBadRequest
		}

	default:
		if len(response.Body) > 0 {
			return response, fmt.Errorf("%w: %d: %s", ErrNon200HttpCode, resp.StatusCode, respBody)
		} else {
			return response, fmt.Errorf("%w: %d", ErrNon200HttpCode, resp.StatusCode)
		}
	}
}

func (c *ClientImpl) do(req *http.Request) (*Response, error) {
	requestTimeTracker := timetrack.New()

	ctx := req.Context()

	log := c.log.GetLogger(ctx)

	requestUrl := req.URL.String()

	baseLogger := func(isSuccess bool, response *Response) logger.Logger {
		requestTimeTracker.Finish()

		fields := map[string]any{
			"logger":         "http_client",
			"method":         req.Method,
			"url":            requestUrl,
			"success":        isSuccess,
			"execution_time": requestTimeTracker.Duration().Milliseconds(),
		}

		if response != nil {
			if value, ok := response.Headers["Content-Type"]; ok {
				fields["content_type"] = strings.Join(value, ",")
			}

			fields["body_bytes"] = len(response.Body)
			fields["status_code"] = response.StatusCode
		}

		return log.WithFields(fields)
	}

	response, err := c.performRequest(ctx, req)
	if err != nil {
		baseLogger(false, response).WithError(err).Error(requestUrl)
	} else {
		baseLogger(true, response).Debug(requestUrl)
	}

	return response, err
}
