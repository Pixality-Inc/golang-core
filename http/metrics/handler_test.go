package metrics_test

import (
	"testing"

	dto "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
	"google.golang.org/protobuf/proto"

	"github.com/pixality-inc/golang-core/errors"
	"github.com/pixality-inc/golang-core/http/metrics"
	coremetrics "github.com/pixality-inc/golang-core/metrics"
)

var errGather = errors.New("test.gather", "gather failed")

type fakeManager struct {
	coremetrics.Manager

	families []*dto.MetricFamily
	err      error
}

func (m *fakeManager) Gather() ([]*dto.MetricFamily, error) {
	return m.families, m.err
}

func TestHandlerGetMetrics(t *testing.T) {
	t.Parallel()

	family := &dto.MetricFamily{
		Name: new("test_counter"),
		Type: dto.MetricType_COUNTER.Enum(),
		Metric: []*dto.Metric{
			{Counter: &dto.Counter{Value: proto.Float64(7)}},
		},
	}

	handler := metrics.NewHandler(&fakeManager{families: []*dto.MetricFamily{family}})

	var ctx fasthttp.RequestCtx

	handler.GetMetrics(&ctx)

	require.Equal(t, fasthttp.StatusOK, ctx.Response.StatusCode())
	assert.Contains(t, string(ctx.Response.Header.ContentType()), "text/plain")
	assert.Contains(t, string(ctx.Response.Body()), "test_counter 7")
}

func TestHandlerGetMetricsError(t *testing.T) {
	t.Parallel()

	handler := metrics.NewHandler(&fakeManager{err: errGather})

	var ctx fasthttp.RequestCtx

	handler.GetMetrics(&ctx)

	require.Equal(t, fasthttp.StatusInternalServerError, ctx.Response.StatusCode())
	assert.Equal(t, "Error gathering metrics", string(ctx.Response.Body()))
}
