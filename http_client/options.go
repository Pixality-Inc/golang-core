package http_client

import (
	"maps"

	"github.com/pixality-inc/golang-core/circuit_breaker"
	"github.com/pixality-inc/golang-core/json"
	"github.com/pixality-inc/golang-core/logger"
)

var jsonBodyLogger = logger.NewLoggableImplWithService("http_client")

// RequestConfig configuration for individual http requests
type RequestConfig struct {
	QueryParams QueryParams
	Headers     Headers
	Body        []byte
	FormData    FormData
}

// RequestOption option for configuring individual http requests
type RequestOption func(*RequestConfig)

// ClientConfig configuration for http client constructor
type ClientConfig struct {
	CircuitBreaker circuit_breaker.CircuitBreaker
}

// Option option for configuring http client
type Option func(*ClientConfig)

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

// WithCircuitBreaker sets custom circuit breaker for http client
func WithCircuitBreaker(cb circuit_breaker.CircuitBreaker) Option {
	return func(cfg *ClientConfig) {
		cfg.CircuitBreaker = cb
	}
}

func applyClientOptions(opts ...Option) *ClientConfig {
	cfg := &ClientConfig{}

	for _, opt := range opts {
		if opt != nil {
			opt(cfg)
		}
	}

	return cfg
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
