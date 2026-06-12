package http

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

var errRender = errors.New("render test error")

type testProtoRenderer struct{}

func (r *testProtoRenderer) Ok() proto.Message {
	return wrapperspb.String("ok")
}

func (r *testProtoRenderer) Error(statusCode int, err error) proto.Message {
	return wrapperspb.String(fmt.Sprintf("%d: %v", statusCode, err))
}

func newRenderCtx(accept string) *fasthttp.RequestCtx {
	ctx := &fasthttp.RequestCtx{}
	if accept != "" {
		ctx.Request.Header.Set("Accept", accept)
	}

	return ctx
}

func TestResponseRendererOk(t *testing.T) {
	t.Parallel()

	renderer := NewResponseRenderer(&testProtoRenderer{})
	ctx := newRenderCtx(mediaTypeJSON)

	renderer.Ok(ctx, wrapperspb.String("hello"))

	assert.Equal(t, fasthttp.StatusOK, ctx.Response.StatusCode())
	assert.Contains(t, string(ctx.Response.Header.ContentType()), mediaTypeJSON)
	assert.JSONEq(t, `"hello"`, string(ctx.Response.Body()))
}

func TestResponseRendererCreated(t *testing.T) {
	t.Parallel()

	renderer := NewResponseRenderer(&testProtoRenderer{})
	ctx := newRenderCtx(mediaTypeJSON)

	renderer.Created(ctx, wrapperspb.String("created"))

	assert.Equal(t, fasthttp.StatusCreated, ctx.Response.StatusCode())
	assert.JSONEq(t, `"created"`, string(ctx.Response.Body()))
}

func TestResponseRendererEmptyOk(t *testing.T) {
	t.Parallel()

	renderer := NewResponseRenderer(&testProtoRenderer{})
	ctx := newRenderCtx(mediaTypeJSON)

	renderer.EmptyOk(ctx)

	assert.Equal(t, fasthttp.StatusOK, ctx.Response.StatusCode())
	assert.JSONEq(t, `"ok"`, string(ctx.Response.Body()))
}

func TestResponseRendererErrorMapping(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		err      error
		expected int
	}{
		{"bad request", ErrBadRequest, fasthttp.StatusBadRequest},
		{"not found", ErrNotFound, fasthttp.StatusNotFound},
		{"unauthorized", ErrUnauthorized, fasthttp.StatusUnauthorized},
		{"forbidden", ErrForbidden, fasthttp.StatusForbidden},
		{"internal", ErrInternalServerError, fasthttp.StatusInternalServerError},
		{"unknown error", errRender, fasthttp.StatusInternalServerError},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			renderer := NewResponseRenderer(&testProtoRenderer{})
			ctx := newRenderCtx(mediaTypeJSON)

			renderer.Error(ctx, testCase.err)

			assert.Equal(t, testCase.expected, ctx.Response.StatusCode())
			assert.Equal(t, testCase.err, ctx.UserValue(RequestMetadataErrorValueKey))
		})
	}
}

func TestResponseRendererErrorHelpers(t *testing.T) {
	t.Parallel()

	renderer := NewResponseRenderer(&testProtoRenderer{})

	testCases := []struct {
		render   func(ctx *fasthttp.RequestCtx)
		expected int
	}{
		{func(ctx *fasthttp.RequestCtx) { renderer.BadRequest(ctx, errRender) }, fasthttp.StatusBadRequest},
		{func(ctx *fasthttp.RequestCtx) { renderer.NotFound(ctx, errRender) }, fasthttp.StatusNotFound},
		{func(ctx *fasthttp.RequestCtx) { renderer.Unauthorized(ctx, errRender) }, fasthttp.StatusUnauthorized},
		{func(ctx *fasthttp.RequestCtx) { renderer.Forbidden(ctx, errRender) }, fasthttp.StatusForbidden},
		{func(ctx *fasthttp.RequestCtx) { renderer.InternalServerError(ctx, errRender) }, fasthttp.StatusInternalServerError},
	}

	for _, testCase := range testCases {
		ctx := newRenderCtx(mediaTypeJSON)

		testCase.render(ctx)

		assert.Equal(t, testCase.expected, ctx.Response.StatusCode())
	}
}

func TestRenderResponseProtobuf(t *testing.T) {
	t.Parallel()

	renderer := NewResponseRenderer(&testProtoRenderer{})
	ctx := newRenderCtx(mediaTypeProtobuf)

	renderer.Ok(ctx, wrapperspb.String("hello"))

	assert.Equal(t, mediaTypeProtobuf, string(ctx.Response.Header.ContentType()))

	var decoded wrapperspb.StringValue

	require.NoError(t, proto.Unmarshal(ctx.Response.Body(), &decoded))
	assert.Equal(t, "hello", decoded.GetValue())
}

func TestRenderResponseUnsupportedAccept(t *testing.T) {
	t.Parallel()

	renderer := NewResponseRenderer(&testProtoRenderer{})
	ctx := newRenderCtx("application/xml")

	renderer.Ok(ctx, wrapperspb.String("hello"))

	assert.Empty(t, ctx.Response.Body())
}

func TestReadBodyJson(t *testing.T) {
	t.Parallel()

	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetContentType(mediaTypeJSON)
	ctx.Request.SetBodyString(`"hello"`)

	var message wrapperspb.StringValue

	require.NoError(t, ReadBody(ctx, &message))
	assert.Equal(t, "hello", message.GetValue())
}

func TestReadBodyProtobuf(t *testing.T) {
	t.Parallel()

	body, err := proto.Marshal(wrapperspb.String("hello"))
	require.NoError(t, err)

	for _, contentType := range []string{mediaTypeProtobuf, mediaTypeXProtobuf} {
		ctx := &fasthttp.RequestCtx{}
		ctx.Request.Header.SetContentType(contentType)
		ctx.Request.SetBody(body)

		var message wrapperspb.StringValue

		require.NoError(t, ReadBody(ctx, &message))
		assert.Equal(t, "hello", message.GetValue(), contentType)
	}
}

func TestReadBodyUnknownContentType(t *testing.T) {
	t.Parallel()

	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetContentType("application/xml")
	ctx.Request.SetBodyString("<xml/>")

	var message wrapperspb.StringValue

	require.Error(t, ReadBody(ctx, &message))
}

func TestReadBodyInvalidJson(t *testing.T) {
	t.Parallel()

	ctx := &fasthttp.RequestCtx{}
	ctx.Request.Header.SetContentType(mediaTypeJSON)
	ctx.Request.SetBodyString("{invalid")

	var message wrapperspb.StringValue

	require.Error(t, ReadBody(ctx, &message))
}

func runWithRenderer(handler fasthttp.RequestHandler) *fasthttp.RequestCtx {
	renderer := NewResponseRenderer(&testProtoRenderer{})
	ctx := newRenderCtx(mediaTypeJSON)

	NewResponseRendererMiddleware(renderer).Handle(handler)(ctx)

	return ctx
}

func TestPackageLevelHelpers(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		handler  fasthttp.RequestHandler
		expected int
	}{
		{"ok", func(ctx *fasthttp.RequestCtx) { Ok(ctx, wrapperspb.String("x")) }, fasthttp.StatusOK},
		{"created", func(ctx *fasthttp.RequestCtx) { Created(ctx, wrapperspb.String("x")) }, fasthttp.StatusCreated},
		{"empty ok", EmptyOk, fasthttp.StatusOK},
		{"bad request", func(ctx *fasthttp.RequestCtx) { BadRequest(ctx) }, fasthttp.StatusBadRequest},
		{"bad request with err", func(ctx *fasthttp.RequestCtx) { BadRequest(ctx, errRender) }, fasthttp.StatusBadRequest},
		{"not found", func(ctx *fasthttp.RequestCtx) { NotFound(ctx) }, fasthttp.StatusNotFound},
		{"not found with err", func(ctx *fasthttp.RequestCtx) { NotFound(ctx, errRender) }, fasthttp.StatusNotFound},
		{"unauthorized", func(ctx *fasthttp.RequestCtx) { Unauthorized(ctx) }, fasthttp.StatusUnauthorized},
		{"unauthorized with err", func(ctx *fasthttp.RequestCtx) { Unauthorized(ctx, errRender) }, fasthttp.StatusUnauthorized},
		{"forbidden", func(ctx *fasthttp.RequestCtx) { Forbidden(ctx) }, fasthttp.StatusForbidden},
		{"forbidden with err", func(ctx *fasthttp.RequestCtx) { Forbidden(ctx, errRender) }, fasthttp.StatusForbidden},
		{"internal", func(ctx *fasthttp.RequestCtx) { InternalServerError(ctx) }, fasthttp.StatusInternalServerError},
		{"internal with err", func(ctx *fasthttp.RequestCtx) { InternalServerError(ctx, errRender) }, fasthttp.StatusInternalServerError},
		{"error", func(ctx *fasthttp.RequestCtx) { Error(ctx, errRender) }, fasthttp.StatusInternalServerError},
		{"handle error", func(ctx *fasthttp.RequestCtx) { HandleError(ctx, ErrNotFound) }, fasthttp.StatusNotFound},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := runWithRenderer(testCase.handler)
			assert.Equal(t, testCase.expected, ctx.Response.StatusCode())
		})
	}
}

func TestPackageLevelHelpersWithoutRenderer(t *testing.T) {
	t.Parallel()

	ctx := newRenderCtx(mediaTypeJSON)

	Ok(ctx, wrapperspb.String("x"))

	assert.Equal(t, fasthttp.StatusOK, ctx.Response.StatusCode())
	assert.Empty(t, ctx.Response.Body())
}

func TestHandleHttpWithModel(t *testing.T) {
	t.Parallel()

	model := wrapperspb.String("entity")

	ctx := runWithRenderer(func(ctx *fasthttp.RequestCtx) {
		HandleHttp(ctx, NewHttpResponseWithModel(&model))
	})

	assert.Equal(t, fasthttp.StatusOK, ctx.Response.StatusCode())
	assert.JSONEq(t, `"entity"`, string(ctx.Response.Body()))
}

func TestHandleHttpWithModelAndOptions(t *testing.T) {
	t.Parallel()

	model := wrapperspb.String("entity")

	ctx := runWithRenderer(func(ctx *fasthttp.RequestCtx) {
		HandleHttp(ctx, NewHttpResponseWithModel(&model, WithHeader("X-Custom", "value")))
	})

	assert.Equal(t, fasthttp.StatusOK, ctx.Response.StatusCode())
	assert.Equal(t, "value", string(ctx.Response.Header.Peek("X-Custom")))
}

func TestHandleHttpWithOptionsOnly(t *testing.T) {
	t.Parallel()

	ctx := runWithRenderer(func(ctx *fasthttp.RequestCtx) {
		HandleHttp(ctx, NewHttpResponseWithOptions[*wrapperspb.StringValue](
			WithStatusCode(fasthttp.StatusTeapot),
			WithBody([]byte("teapot")),
		))
	})

	assert.Equal(t, fasthttp.StatusTeapot, ctx.Response.StatusCode())
	assert.Equal(t, "teapot", string(ctx.Response.Body()))
}

func TestHandleHttpEmptyResponse(t *testing.T) {
	t.Parallel()

	ctx := runWithRenderer(func(ctx *fasthttp.RequestCtx) {
		HandleHttp(ctx, NewHttpResponseWithOptions[*wrapperspb.StringValue]())
	})

	assert.Equal(t, fasthttp.StatusInternalServerError, ctx.Response.StatusCode())
}

func TestHttpResponseOptions(t *testing.T) {
	t.Parallel()

	var ctx fasthttp.RequestCtx

	WithHeader("X-A", "1")(&ctx)
	WithHeaders(map[string]string{"X-B": "2", "X-C": "3"})(&ctx)
	WithStatusCode(fasthttp.StatusTeapot)(&ctx)
	WithBody([]byte("body"))(&ctx)

	assert.Equal(t, "1", string(ctx.Response.Header.Peek("X-A")))
	assert.Equal(t, "2", string(ctx.Response.Header.Peek("X-B")))
	assert.Equal(t, "3", string(ctx.Response.Header.Peek("X-C")))
	assert.Equal(t, fasthttp.StatusTeapot, ctx.Response.StatusCode())
	assert.Equal(t, "body", string(ctx.Response.Body()))
}

func TestWithRedirect(t *testing.T) {
	t.Parallel()

	var ctx fasthttp.RequestCtx

	ctx.Request.SetRequestURI("http://example.com/from")

	WithRedirect("/to", fasthttp.StatusMovedPermanently)(&ctx)

	assert.Equal(t, fasthttp.StatusMovedPermanently, ctx.Response.StatusCode())
	assert.Equal(t, "http://example.com/to", string(ctx.Response.Header.Peek("Location")))
}

func TestNewHttpResponseWithModelDefaults(t *testing.T) {
	t.Parallel()

	model := wrapperspb.String("entity")
	response := NewHttpResponseWithModel(&model)

	assert.Same(t, &model, response.Model())
	assert.NotNil(t, response.Options())
	assert.Empty(t, response.Options())
}
