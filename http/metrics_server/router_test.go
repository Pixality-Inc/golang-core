package metrics_server_test

import (
	"testing"

	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/valyala/fasthttp"

	"github.com/pixality-inc/golang-core/http/metrics"
	"github.com/pixality-inc/golang-core/http/metrics_server"
	coremetrics "github.com/pixality-inc/golang-core/metrics"
)

type fakeManager struct {
	coremetrics.Manager
}

func (m *fakeManager) Gather() ([]*dto.MetricFamily, error) {
	return nil, nil
}

func TestNewRouter(t *testing.T) {
	t.Parallel()

	handler := metrics_server.NewRouter(metrics.NewHandler(&fakeManager{})).Handle()

	var ctx fasthttp.RequestCtx

	ctx.Request.SetRequestURI("/metrics")
	ctx.Request.Header.SetMethod(fasthttp.MethodGet)

	handler(&ctx)

	assert.Equal(t, fasthttp.StatusOK, ctx.Response.StatusCode())
}
