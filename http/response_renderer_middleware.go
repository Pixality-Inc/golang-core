package http

import (
	"context"

	"github.com/valyala/fasthttp"
)

const ResponseRendererValueKey = "response_renderer"

type ResponseRendererMiddleware struct {
	responseRenderer ResponseRenderer
}

func NewResponseRendererMiddleware(responseRenderer ResponseRenderer) *ResponseRendererMiddleware {
	return &ResponseRendererMiddleware{
		responseRenderer: responseRenderer,
	}
}

func (m *ResponseRendererMiddleware) Handle(next fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		m.handle(ctx, next)
	}
}

func (m *ResponseRendererMiddleware) handle(ctx *fasthttp.RequestCtx, next fasthttp.RequestHandler) {
	ctx.SetUserValue(ResponseRendererValueKey, m.responseRenderer)

	next(ctx)
}

func GetResponseRenderer(ctx context.Context) ResponseRenderer {
	responseRenderer, ok := ctx.Value(ResponseRendererValueKey).(ResponseRenderer)
	if !ok {
		return nil
	}

	return responseRenderer
}
