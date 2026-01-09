package http

import (
	"google.golang.org/protobuf/proto"
)

type RedirectOption struct {
	Url    string
	Status int
}

func NewRedirectOption(url string, status int) *RedirectOption {
	return &RedirectOption{
		Url:    url,
		Status: status,
	}
}

type HttpResponseOptions struct {
	Redirect *RedirectOption
}

func NewHttpResponseOptions() *HttpResponseOptions {
	return &HttpResponseOptions{
		Redirect: nil,
	}
}

func (o *HttpResponseOptions) WithRedirect(redirect *RedirectOption) *HttpResponseOptions {
	o.Redirect = redirect

	return o
}

type HttpResponse[T proto.Message] interface {
	Model() *T
	Options() *HttpResponseOptions
}

type HttpResponseImpl[T proto.Message] struct {
	model   *T
	options *HttpResponseOptions
}

func (r *HttpResponseImpl[T]) Model() *T {
	return r.model
}

func (r *HttpResponseImpl[T]) Options() *HttpResponseOptions {
	return r.options
}

func NewHttpResponseWithModel[T proto.Message](model *T) HttpResponse[T] {
	return &HttpResponseImpl[T]{
		model: model,
	}
}

func NewHttpResponseWithOptions[T proto.Message](options *HttpResponseOptions) HttpResponse[T] {
	return &HttpResponseImpl[T]{
		options: options,
	}
}
