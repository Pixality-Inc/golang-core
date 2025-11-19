package http

import (
	"strings"

	"github.com/pixality-inc/golang-core/logger"

	"github.com/valyala/fasthttp"
)

const (
	AccessControlAllowOriginHeader      = "Access-Control-Allow-Origin"
	AccessControlAllowHeadersHeader     = "Access-Control-Allow-Headers"
	AccessControlAllowMethodsHeader     = "Access-Control-Allow-Methods"
	AccessControlAllowCredentialsHeader = "Access-Control-Allow-Credentials"
)

type CorsMiddleware struct {
	log          logger.Loggable
	origin       string
	extraHeaders []string
}

func NewCorsMiddleware(origin string, extraHeaders ...string) *CorsMiddleware {
	return &CorsMiddleware{
		log:          logger.NewLoggableImplWithService("cors_middleware"),
		origin:       origin,
		extraHeaders: extraHeaders,
	}
}

func (m *CorsMiddleware) Handle(next fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		if ctx.IsOptions() {
			m.addHeaders(ctx)

			EmptyOk(ctx)

			return
		}

		m.handle(ctx, next)
	}
}

func (m *CorsMiddleware) handle(ctx *fasthttp.RequestCtx, next fasthttp.RequestHandler) {
	m.addHeaders(ctx)

	next(ctx)
}

func (m *CorsMiddleware) addHeaders(ctx *fasthttp.RequestCtx) {
	headers := []string{
		"Authorization",
		"Content-Type",
		"Accept",
		"Content-Encoding",
		"Upgrade",
		"User-Agent",
	}

	headers = append(headers, m.extraHeaders...)

	headersStr := strings.Join(headers, ",")

	ctx.Response.Header.Add(AccessControlAllowOriginHeader, m.origin)
	ctx.Response.Header.Add(AccessControlAllowHeadersHeader, headersStr)
	ctx.Response.Header.Add(AccessControlAllowMethodsHeader, "GET,HEAD,POST,PUT,PATCH,OPTIONS,DELETE")
	ctx.Response.Header.Add(AccessControlAllowCredentialsHeader, "true")
}
