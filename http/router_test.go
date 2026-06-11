package http

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"
)

func TestRouterDispatch(t *testing.T) {
	t.Parallel()

	router := NewRouter()

	router.GET("/res", func(ctx *fasthttp.RequestCtx) { ctx.SetBodyString("get") })
	router.POST("/res", func(ctx *fasthttp.RequestCtx) { ctx.SetBodyString("post") })
	router.DELETE("/res", func(ctx *fasthttp.RequestCtx) { ctx.SetBodyString("delete") })
	router.OPTIONS("/res", func(ctx *fasthttp.RequestCtx) { ctx.SetBodyString("options") })

	handler := router.Handle()

	testCases := []struct {
		method   string
		expected string
	}{
		{fasthttp.MethodGet, "get"},
		{fasthttp.MethodPost, "post"},
		{fasthttp.MethodDelete, "delete"},
		{fasthttp.MethodOptions, "options"},
	}

	for _, testCase := range testCases {
		var ctx fasthttp.RequestCtx

		ctx.Request.SetRequestURI("/res")
		ctx.Request.Header.SetMethod(testCase.method)

		handler(&ctx)

		assert.Equal(t, testCase.expected, string(ctx.Response.Body()), testCase.method)
	}
}

func TestRouterNotFound(t *testing.T) {
	t.Parallel()

	router := NewRouter()
	router.GET("/res", func(ctx *fasthttp.RequestCtx) { ctx.SetBodyString("get") })

	var ctx fasthttp.RequestCtx

	ctx.Request.SetRequestURI("/missing")
	ctx.Request.Header.SetMethod(fasthttp.MethodGet)

	router.Handle()(&ctx)

	assert.Equal(t, fasthttp.StatusNotFound, ctx.Response.StatusCode())
}
