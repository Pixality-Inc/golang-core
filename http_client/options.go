package http_client

import (
	"maps"

	"github.com/pixality-inc/golang-core/json"
	"github.com/pixality-inc/golang-core/logger"
)

var jsonBodyLogger = logger.NewLoggableImplWithService("http_client")

type RequestConfig struct {
	QueryParams QueryParams
	Headers     Headers
	Body        []byte
	FormData    FormData
}

type RequestOption func(*RequestConfig)

func WithBody(body []byte) RequestOption {
	return func(cfg *RequestConfig) {
		cfg.Body = body
	}
}

func WithJsonBody(v any) RequestOption {
	return func(cfg *RequestConfig) {
		data, err := json.Marshal(v)
		if err != nil {
			jsonBodyLogger.
				GetLoggerWithoutContext().
				WithError(err).
				Error("failed to marshal json body")

			return
		}

		cfg.Body = data
	}
}

func WithFormData(data FormData) RequestOption {
	return func(cfg *RequestConfig) {
		cfg.FormData = data
	}
}

func WithHeader(key, value string) RequestOption {
	return func(cfg *RequestConfig) {
		if cfg.Headers == nil {
			cfg.Headers = make(Headers)
		}

		cfg.Headers[key] = append(cfg.Headers[key], value)
	}
}

func WithHeaders(headers Headers) RequestOption {
	return func(cfg *RequestConfig) {
		if cfg.Headers == nil {
			cfg.Headers = make(Headers)
		}

		for key, values := range headers {
			cfg.Headers[key] = append(cfg.Headers[key], values...)
		}
	}
}

func WithQueryParam(key, value string) RequestOption {
	return func(cfg *RequestConfig) {
		if cfg.QueryParams == nil {
			cfg.QueryParams = make(QueryParams)
		}

		cfg.QueryParams[key] = value
	}
}

func WithQueryParams(params QueryParams) RequestOption {
	return func(cfg *RequestConfig) {
		if cfg.QueryParams == nil {
			cfg.QueryParams = make(QueryParams)
		}

		maps.Copy(cfg.QueryParams, params)
	}
}

func applyOptions(opts ...RequestOption) *RequestConfig {
	cfg := &RequestConfig{
		QueryParams: make(QueryParams),
		Headers:     make(Headers),
	}

	for _, opt := range opts {
		if opt != nil {
			opt(cfg)
		}
	}

	return cfg
}
