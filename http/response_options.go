package http

import (
	"github.com/valyala/fasthttp"
	"google.golang.org/protobuf/proto"
)

type HttpResponseOption = func(ctx *fasthttp.RequestCtx)

func WithRedirect(url string, status int) HttpResponseOption {
	return func(ctx *fasthttp.RequestCtx) {
		ctx.Redirect(url, status)
	}
}

func WithHeader(key string, value string) HttpResponseOption {
	return func(ctx *fasthttp.RequestCtx) {
		ctx.Response.Header.Set(key, value)
	}
}

func WithHeaders(headers map[string]string) HttpResponseOption {
	return func(ctx *fasthttp.RequestCtx) {
		for key, value := range headers {
			ctx.Response.Header.Set(key, value)
		}
	}
}

func WithStatusCode(statusCode int) HttpResponseOption {
	return func(ctx *fasthttp.RequestCtx) {
		ctx.SetStatusCode(statusCode)
	}
}

func WithBody(body []byte) HttpResponseOption {
	return func(ctx *fasthttp.RequestCtx) {
		ctx.Response.SetBody(body)
	}
}

type HttpResponse[T proto.Message] interface {
	Model() *T
	Options() []HttpResponseOption
}

type HttpResponseImpl[T proto.Message] struct {
	model   *T
	options []HttpResponseOption
}

func (r *HttpResponseImpl[T]) Model() *T {
	return r.model
}

func (r *HttpResponseImpl[T]) Options() []HttpResponseOption {
	return r.options
}

func NewHttpResponseWithModel[T proto.Message](model *T, options ...HttpResponseOption) HttpResponse[T] {
	if options == nil {
		options = make([]HttpResponseOption, 0)
	}

	return &HttpResponseImpl[T]{
		model:   model,
		options: options,
	}
}

func NewHttpResponseWithOptions[T proto.Message](options ...HttpResponseOption) HttpResponse[T] {
	return &HttpResponseImpl[T]{
		options: options,
	}
}
