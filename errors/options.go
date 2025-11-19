package errors

import "maps"

type Option interface {
	Apply(errorImpl *ImplError)
}

type withCause struct {
	cause error
}

func WithCause(cause error) Option {
	return &withCause{
		cause: cause,
	}
}

func (e *withCause) Apply(errorImpl *ImplError) {
	errorImpl.cause = e.cause
}

type withParam struct {
	key   string
	value any
}

func WithParam(key string, value any) Option {
	return &withParam{
		key: key,
	}
}

func (e *withParam) Apply(errorImpl *ImplError) {
	errorImpl.params[e.key] = e.value
}

type withParams struct {
	params Params
}

func WithParams(params Params) Option {
	return &withParams{
		params: params,
	}
}

func (e *withParams) Apply(errorImpl *ImplError) {
	maps.Copy(errorImpl.params, e.params)
}
