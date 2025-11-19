package http

import (
	"github.com/valyala/fasthttp"
)

type Middleware func(fasthttp.RequestHandler) fasthttp.RequestHandler

func (f Middleware) Handle(next fasthttp.RequestHandler) fasthttp.RequestHandler {
	return f(next)
}
