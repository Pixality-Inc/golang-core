package http

import (
	router2 "github.com/fasthttp/router"
	"github.com/valyala/fasthttp"
)

type Router interface {
	OPTIONS(path string, handle fasthttp.RequestHandler)
	GET(path string, handle fasthttp.RequestHandler)
	POST(path string, handle fasthttp.RequestHandler)
	DELETE(path string, handle fasthttp.RequestHandler)
	Handle() fasthttp.RequestHandler
}

type RouterImpl struct {
	router *router2.Router
}

func NewRouter() Router {
	return &RouterImpl{
		router: router2.New(),
	}
}

func (r *RouterImpl) OPTIONS(path string, handle fasthttp.RequestHandler) {
	r.router.OPTIONS(path, handle)
}

func (r *RouterImpl) GET(path string, handle fasthttp.RequestHandler) {
	r.router.GET(path, handle)
}

func (r *RouterImpl) POST(path string, handle fasthttp.RequestHandler) {
	r.router.POST(path, handle)
}

func (r *RouterImpl) DELETE(path string, handle fasthttp.RequestHandler) {
	r.router.DELETE(path, handle)
}

func (r *RouterImpl) Handle() fasthttp.RequestHandler {
	return r.router.Handler
}
