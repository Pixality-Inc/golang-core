package healthcheck_test

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"

	"github.com/pixality-inc/golang-core/http/healthcheck"
	"github.com/pixality-inc/golang-core/logger"
)

type fakeService struct {
	ok atomic.Bool
}

func (s *fakeService) IsOK() bool {
	return s.ok.Load()
}

func readinessStatus(handler *healthcheck.Handler) int {
	var ctx fasthttp.RequestCtx

	handler.GetReadiness(&ctx)

	return ctx.Response.StatusCode()
}

func TestHandlerHealthy(t *testing.T) {
	t.Parallel()

	service := &fakeService{}
	service.ok.Store(true)

	handler := healthcheck.NewDefaultHandler(t.Context(), time.Hour, service)

	assert.Equal(t, fasthttp.StatusOK, readinessStatus(handler))
}

func TestHandlerUnhealthy(t *testing.T) {
	t.Parallel()

	service := &fakeService{}

	handler := healthcheck.NewHandler(t.Context(), healthcheck.Options{
		ReCheckAfter: time.Hour,
		Logger:       logger.NewLoggableImplWithService("test").GetLoggerWithoutContext(),
	}, healthcheck.Named("db", service), service)

	assert.Equal(t, fasthttp.StatusServiceUnavailable, readinessStatus(handler))
}

func TestHandlerNoServices(t *testing.T) {
	t.Parallel()

	handler := healthcheck.NewDefaultHandler(t.Context(), time.Hour)

	assert.Equal(t, fasthttp.StatusOK, readinessStatus(handler))
}

func TestHandlerRecovers(t *testing.T) {
	t.Parallel()

	service := &fakeService{}

	handler := healthcheck.NewDefaultHandler(t.Context(), 5*time.Millisecond, service)

	require.Equal(t, fasthttp.StatusServiceUnavailable, readinessStatus(handler))

	service.ok.Store(true)

	require.Eventually(t, func() bool {
		return readinessStatus(handler) == fasthttp.StatusOK
	}, 3*time.Second, 5*time.Millisecond)
}

func TestHandlerDegrades(t *testing.T) {
	t.Parallel()

	service := &fakeService{}
	service.ok.Store(true)

	handler := healthcheck.NewDefaultHandler(t.Context(), 5*time.Millisecond, service)

	require.Equal(t, fasthttp.StatusOK, readinessStatus(handler))

	service.ok.Store(false)

	require.Eventually(t, func() bool {
		return readinessStatus(handler) == fasthttp.StatusServiceUnavailable
	}, 3*time.Second, 5*time.Millisecond)
}

func TestNamedService(t *testing.T) {
	t.Parallel()

	service := &fakeService{}
	service.ok.Store(true)

	named := healthcheck.Named("redis", service)

	assert.Equal(t, "redis", named.Name())
	assert.True(t, named.IsOK())
}
