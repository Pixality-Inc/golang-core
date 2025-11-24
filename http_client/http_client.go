package http_client

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/valyala/fasthttp"

	http2 "github.com/pixality-inc/golang-core/http"
	"github.com/pixality-inc/golang-core/json"
	"github.com/pixality-inc/golang-core/logger"
	"github.com/pixality-inc/golang-core/retry"
	"github.com/pixality-inc/golang-core/timetrack"
)

var (
	ErrNotFound            = errors.New("not found")
	ErrBadRequest          = errors.New("bad request")
	ErrNon200HttpCode      = errors.New("non-200 http status code")
	ErrInvalidFormDataType = errors.New("invalid form data type")
	ErrTLSConfig           = errors.New("failed to configure tls")
)

type Client interface {
	Get(ctx context.Context, uri string, opts ...RequestOption) (Response, error)
	Post(ctx context.Context, uri string, opts ...RequestOption) (Response, error)
	Put(ctx context.Context, uri string, opts ...RequestOption) (Response, error)
	Patch(ctx context.Context, uri string, opts ...RequestOption) (Response, error)
	Delete(ctx context.Context, uri string, opts ...RequestOption) (Response, error)
	Head(ctx context.Context, uri string, opts ...RequestOption) (Response, error)
	Options(ctx context.Context, uri string, opts ...RequestOption) (Response, error)

	Do(ctx context.Context, method, uri string, opts ...RequestOption) (Response, error)
}

type ClientImpl struct {
	log    logger.Loggable
	config Config
	client *fasthttp.Client
}

func NewClientImpl(
	log logger.Loggable,
	config Config,
) (*ClientImpl, error) {
	readTimeout := config.ReadTimeout()
	if readTimeout == 0 {
		readTimeout = config.Timeout()
	}

	writeTimeout := config.WriteTimeout()
	if writeTimeout == 0 {
		writeTimeout = config.Timeout()
	}

	client := &fasthttp.Client{
		Name:                     config.Name(),
		ReadTimeout:              readTimeout,
		WriteTimeout:             writeTimeout,
		MaxConnsPerHost:          config.MaxConnsPerHost(),
		MaxIdleConnDuration:      config.MaxIdleConnDuration(),
		MaxConnWaitTimeout:       config.MaxConnWaitTimeout(),
		MaxConnDuration:          config.MaxConnDuration(),
		ReadBufferSize:           config.ReadBufferSize(),
		WriteBufferSize:          config.WriteBufferSize(),
		MaxResponseBodySize:      config.MaxResponseBodySize(),
		NoDefaultUserAgentHeader: false,
		DisablePathNormalizing:   false,
		StreamResponseBody:       config.StreamResponseBody(),
	}

	tlsConfig, err := configureTLS(config)
	if err != nil {
		return nil, err
	}

	client.TLSConfig = tlsConfig

	return &ClientImpl{
		log:    log,
		config: config,
		client: client,
	}, nil
}

func hasTLSConfig(config Config) bool {
	return config.InsecureSkipVerify() ||
		config.TLSMinVersion() != 0 ||
		config.TLSMaxVersion() != 0 ||
		config.TLSServerName() != "" ||
		config.TLSRootCAFile() != "" ||
		config.TLSClientCertFile() != "" ||
		config.TLSClientKeyFile() != ""
}

func configureTLS(config Config) (*tls.Config, error) {
	if !hasTLSConfig(config) {
		return nil, nil
	}

	tlsConfig := &tls.Config{
		InsecureSkipVerify: config.InsecureSkipVerify(), // nolint:gosec
	}

	if config.TLSMinVersion() != 0 {
		tlsConfig.MinVersion = config.TLSMinVersion()
	}

	if config.TLSMaxVersion() != 0 {
		tlsConfig.MaxVersion = config.TLSMaxVersion()
	}

	if config.TLSServerName() != "" {
		tlsConfig.ServerName = config.TLSServerName()
	}

	if err := loadRootCA(config, tlsConfig); err != nil {
		return nil, err
	}

	if err := loadClientCertificate(config, tlsConfig); err != nil {
		return nil, err
	}

	return tlsConfig, nil
}

func loadRootCA(config Config, tlsConfig *tls.Config) error {
	if config.TLSRootCAFile() == "" {
		return nil
	}

	caCert, err := os.ReadFile(config.TLSRootCAFile())
	if err != nil {
		return fmt.Errorf("%w: failed to read root ca file: %w", ErrTLSConfig, err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
		return fmt.Errorf("%w: failed to parse root ca certificate", ErrTLSConfig)
	}

	tlsConfig.RootCAs = caCertPool

	return nil
}

func loadClientCertificate(config Config, tlsConfig *tls.Config) error {
	if config.TLSClientCertFile() == "" || config.TLSClientKeyFile() == "" {
		return nil
	}

	cert, err := tls.LoadX509KeyPair(config.TLSClientCertFile(), config.TLSClientKeyFile())
	if err != nil {
		return fmt.Errorf("%w: failed to load client certificate: %w", ErrTLSConfig, err)
	}

	tlsConfig.Certificates = []tls.Certificate{cert}

	return nil
}

func (c *ClientImpl) Get(ctx context.Context, uri string, opts ...RequestOption) (Response, error) {
	return c.Do(ctx, http.MethodGet, uri, opts...)
}

func (c *ClientImpl) Post(ctx context.Context, uri string, opts ...RequestOption) (Response, error) {
	return c.Do(ctx, http.MethodPost, uri, opts...)
}

func (c *ClientImpl) Put(ctx context.Context, uri string, opts ...RequestOption) (Response, error) {
	return c.Do(ctx, http.MethodPut, uri, opts...)
}

func (c *ClientImpl) Patch(ctx context.Context, uri string, opts ...RequestOption) (Response, error) {
	return c.Do(ctx, http.MethodPatch, uri, opts...)
}

func (c *ClientImpl) Delete(ctx context.Context, uri string, opts ...RequestOption) (Response, error) {
	return c.Do(ctx, http.MethodDelete, uri, opts...)
}

func (c *ClientImpl) Head(ctx context.Context, uri string, opts ...RequestOption) (Response, error) {
	return c.Do(ctx, http.MethodHead, uri, opts...)
}

func (c *ClientImpl) Options(ctx context.Context, uri string, opts ...RequestOption) (Response, error) {
	return c.Do(ctx, http.MethodOptions, uri, opts...)
}

func (c *ClientImpl) Do(ctx context.Context, method, uri string, opts ...RequestOption) (Response, error) {
	cfg := applyOptions(opts...)

	if c.config.RetryPolicy() != nil {
		return retry.DoWithCondition(
			ctx,
			c.config.RetryPolicy(),
			c.log,
			func() (Response, error) {
				return c.performRequest(ctx, method, uri, cfg)
			},
			func(response Response, err error) bool {
				statusCode := 0
				if response != nil {
					statusCode = response.GetStatusCode()
				}

				return retry.ShouldRetry(statusCode, err)
			},
		)
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

func AsJson[OUT any](response Response, defaultValue OUT) (TypedResponse[OUT], error) {
	typedResponse := &TypedResponseImpl[OUT]{
		StatusCode: response.GetStatusCode(),
		Headers:    response.GetHeaders(),
		Body:       response.GetBody(),
		Entity:     defaultValue,
	}

	if err := json.Unmarshal(response.GetBody(), &typedResponse.Entity); err != nil {
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
		formData, ok := cfg.FormData.(*FormDataImpl)
		if !ok {
			return ErrInvalidFormDataType
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

// performRequest uses config timeout as fasthttp client timeout
// ctx is checked after request to respect cancellation and deadlines
func (c *ClientImpl) performRequest(ctx context.Context, method, uri string, cfg *RequestConfig) (Response, error) {
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

	response := &ResponseImpl{
		StatusCode: resp.StatusCode(),
		Headers:    make(Headers),
		Body:       make([]byte, len(resp.Body())),
	}

	copy(response.Body, resp.Body())

	resp.Header.VisitAll(func(key, value []byte) { // nolint:staticcheck
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
	response Response,
	err error,
	tracker *timetrack.TimeTracker,
) {
	tracker.Finish()

	log := c.log.GetLogger(ctx)

	fields := map[string]any{
		"logger":         c.config.Name(),
		"method":         method,
		"url":            url,
		"success":        err == nil && (response == nil || (response.GetStatusCode() >= 200 && response.GetStatusCode() < 300)),
		"execution_time": tracker.Duration().Milliseconds(),
	}

	if response != nil {
		if value, ok := response.GetHeaders()["Content-Type"]; ok {
			fields["content_type"] = strings.Join(value, ",")
		}

		fields["body_bytes"] = len(response.GetBody())
		fields["status_code"] = response.GetStatusCode()
	}

	logWithFields := log.WithFields(fields)

	switch {
	case err != nil:
		logWithFields.WithError(err).Error(url)
	case response != nil && (response.GetStatusCode() < 200 || response.GetStatusCode() >= 300):
		logWithFields.Warn(url)
	default:
		logWithFields.Debug(url)
	}
}

func (c *ClientImpl) handleStatusCode(response Response) (Response, error) {
	statusCode := response.GetStatusCode()
	body := response.GetBody()

	switch {
	case statusCode >= 200 && statusCode <= 299:
		return response, nil

	case statusCode == http.StatusNotFound:
		if len(body) > 0 {
			return response, fmt.Errorf("%w: %s", ErrNotFound, body)
		}

		return response, ErrNotFound

	case statusCode == http.StatusBadRequest:
		if len(body) > 0 {
			return response, fmt.Errorf("%w: %s", ErrBadRequest, body)
		}

		return response, ErrBadRequest

	default:
		if len(body) > 0 {
			return response, fmt.Errorf("%w: %d: %s", ErrNon200HttpCode, statusCode, body)
		}

		return response, fmt.Errorf("%w: %d", ErrNon200HttpCode, statusCode)
	}
}
