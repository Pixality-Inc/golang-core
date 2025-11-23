package http_client

import "github.com/pixality-inc/golang-core/json"

type RequestConfig struct {
	QueryParams QueryParams
	Headers     Headers
	Body        []byte
	FormData    FormDataInterface
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
			cfg.Body = nil
			return
		}
		cfg.Body = data
	}
}

func WithFormData(data FormDataInterface) RequestOption {
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
			for _, value := range values {
				cfg.Headers[key] = append(cfg.Headers[key], value)
			}
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
		for key, value := range params {
			cfg.QueryParams[key] = value
		}
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
