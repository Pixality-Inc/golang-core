package http

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"

	"github.com/pixality-inc/golang-core/logger"
)

func TestCorsMiddlewareAddsHeadersAndCallsNext(t *testing.T) {
	t.Parallel()

	middleware := NewCorsMiddleware("https://example.com", "X-Extra")

	var nextCalled bool

	handler := middleware.Handle(func(_ *fasthttp.RequestCtx) {
		nextCalled = true
	})

	var ctx fasthttp.RequestCtx

	ctx.Request.Header.SetMethod(fasthttp.MethodGet)

	handler(&ctx)

	assert.True(t, nextCalled)
	assert.Equal(t, "https://example.com", string(ctx.Response.Header.Peek(AccessControlAllowOriginHeader)))
	assert.Contains(t, string(ctx.Response.Header.Peek(AccessControlAllowHeadersHeader)), "Authorization")
	assert.Contains(t, string(ctx.Response.Header.Peek(AccessControlAllowHeadersHeader)), "X-Extra")
	assert.Contains(t, string(ctx.Response.Header.Peek(AccessControlAllowMethodsHeader)), "DELETE")
	assert.Equal(t, "true", string(ctx.Response.Header.Peek(AccessControlAllowCredentialsHeader)))
}

func TestCorsMiddlewarePreflightSkipsNext(t *testing.T) {
	t.Parallel()

	middleware := NewCorsMiddleware("*")

	var nextCalled bool

	handler := middleware.Handle(func(_ *fasthttp.RequestCtx) {
		nextCalled = true
	})

	var ctx fasthttp.RequestCtx

	ctx.Request.Header.SetMethod(fasthttp.MethodOptions)

	handler(&ctx)

	assert.False(t, nextCalled)
	assert.Equal(t, "*", string(ctx.Response.Header.Peek(AccessControlAllowOriginHeader)))
	assert.Equal(t, fasthttp.StatusOK, ctx.Response.StatusCode())
}

func TestRequestMetadataMiddlewareGeneratesRequestId(t *testing.T) {
	t.Parallel()

	middleware := NewRequestMetadataMiddleware()

	var ctx fasthttp.RequestCtx

	middleware.Handle(func(_ *fasthttp.RequestCtx) {})(&ctx)

	metadata := GetRequestMetadata(&ctx)
	require.NotNil(t, metadata)

	_, err := uuid.Parse(metadata.RequestId)
	require.NoError(t, err)

	assert.Equal(t, metadata.RequestId, ctx.UserValue(RequestIdValueKey))
}

func TestRequestMetadataMiddlewarePreservesRequestId(t *testing.T) {
	t.Parallel()

	middleware := NewRequestMetadataMiddleware()

	var ctx fasthttp.RequestCtx

	ctx.Request.Header.Set("X-Request-Id", "incoming-id")
	ctx.Request.Header.Set("cf-ipcountry", "NL")
	ctx.Request.Header.Set("cf-ray", "ray-1")
	ctx.Request.Header.Set("cf-connecting-ip", "1.2.3.4")

	var nextCalled bool

	middleware.Handle(func(_ *fasthttp.RequestCtx) {
		nextCalled = true
	})(&ctx)

	require.True(t, nextCalled)

	metadata := GetRequestMetadata(&ctx)
	require.NotNil(t, metadata)
	assert.Equal(t, "incoming-id", metadata.RequestId)
	assert.Equal(t, "NL", metadata.CfIpCountry)
	assert.Equal(t, "ray-1", metadata.CfRay)
	assert.Equal(t, "1.2.3.4", metadata.CfConnectingIp)
}

func TestGetRequestMetadataAbsent(t *testing.T) {
	t.Parallel()

	var ctx fasthttp.RequestCtx

	assert.Nil(t, GetRequestMetadata(&ctx))

	ctx.SetUserValue(RequestMetadataValueKey, "wrong type")
	assert.Nil(t, GetRequestMetadata(&ctx))
}

func TestResponseRendererMiddleware(t *testing.T) {
	t.Parallel()

	renderer := NewResponseRenderer(&testProtoRenderer{})
	middleware := NewResponseRendererMiddleware(renderer)

	var ctx fasthttp.RequestCtx

	var nextCalled bool

	middleware.Handle(func(_ *fasthttp.RequestCtx) {
		nextCalled = true
	})(&ctx)

	assert.True(t, nextCalled)
	assert.Equal(t, ResponseRenderer(renderer), GetResponseRenderer(&ctx))
}

func TestGetResponseRendererAbsent(t *testing.T) {
	t.Parallel()

	var ctx fasthttp.RequestCtx

	assert.Nil(t, GetResponseRenderer(&ctx))
}

func TestMiddlewareFunc(t *testing.T) {
	t.Parallel()

	var order []string

	middleware := Middleware(func(next fasthttp.RequestHandler) fasthttp.RequestHandler {
		return func(ctx *fasthttp.RequestCtx) {
			order = append(order, "before")

			next(ctx)
		}
	})

	var ctx fasthttp.RequestCtx

	middleware.Handle(func(_ *fasthttp.RequestCtx) {
		order = append(order, "handler")
	})(&ctx)

	assert.Equal(t, []string{"before", "handler"}, order)
}

func TestAddLogsExpanders(t *testing.T) {
	t.Parallel()

	var ctx fasthttp.RequestCtx

	expander := NewRequestMetadataLogsExpander()

	AddLogsExpanders(&ctx, expander)
	AddLogsExpanders(&ctx, expander)

	expanders, ok := ctx.UserValue(logger.LogsExpandersName).(logger.LogsExpanders)
	require.True(t, ok)
	assert.Len(t, expanders, 2)
}

func TestRequestMetadataLogsExpanderExpand(t *testing.T) {
	t.Parallel()

	expander := NewRequestMetadataLogsExpander()
	log := logger.NewLoggableImplWithService("test").GetLoggerWithoutContext()

	var ctx fasthttp.RequestCtx

	assert.Equal(t, log, expander.Expand(&ctx, log))

	ctx.SetUserValue(RequestMetadataValueKey, &RequestMetadata{RequestId: "rid"})
	assert.NotNil(t, expander.Expand(&ctx, log))
}
