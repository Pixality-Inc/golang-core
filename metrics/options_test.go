package metrics_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/pixality-inc/golang-core/metrics"
)

func TestMetricDescriptionDefaults(t *testing.T) {
	t.Parallel()

	description := metrics.NewMetricDescription("requests_total")

	assert.Equal(t, "requests_total", description.Name())
	assert.Empty(t, description.Namespace())
	assert.Empty(t, description.Subsystem())
	assert.Empty(t, description.Help())
	assert.NotNil(t, description.Labels())
	assert.Empty(t, description.Labels())
}

func TestMetricDescriptionBuilder(t *testing.T) {
	t.Parallel()

	description := metrics.NewMetricDescription("requests_total").
		WithNamespace("app").
		WithSubsystem("http").
		WithHelp("Total requests").
		WithLabel("method", "GET").
		WithLabels(map[string]string{"method": "POST", "status": "200"})

	assert.Equal(t, "requests_total", description.Name())
	assert.Equal(t, "app", description.Namespace())
	assert.Equal(t, "http", description.Subsystem())
	assert.Equal(t, "Total requests", description.Help())
	assert.Equal(t, map[string]string{"method": "POST", "status": "200"}, description.Labels())
}

func TestHistogramOptionsDefaults(t *testing.T) {
	t.Parallel()

	options := metrics.NewHistogramOptions()

	assert.Nil(t, options.Buckets())
	assert.Zero(t, options.NativeHistogramBucketFactor())
	assert.Zero(t, options.NativeHistogramZeroThreshold())
	assert.Zero(t, options.NativeHistogramMaxBucketNumber())
	assert.Zero(t, options.NativeHistogramMinResetDuration())
	assert.Zero(t, options.NativeHistogramMaxZeroThreshold())
	assert.Zero(t, options.NativeHistogramMaxExemplars())
	assert.Zero(t, options.NativeHistogramExemplarTTL())
}

func TestHistogramOptionsBuilder(t *testing.T) {
	t.Parallel()

	options := metrics.NewHistogramOptions().
		WithBuckets([]float64{0.1, 0.5, 1}).
		WithNativeHistogramBucketFactor(1.1).
		WithNativeHistogramZeroThreshold(0.001).
		WithNativeHistogramMaxBucketNumber(160).
		WithNativeHistogramMinResetDuration(time.Hour).
		WithNativeHistogramMaxZeroThreshold(0.01).
		WithNativeHistogramMaxExemplars(10).
		WithNativeHistogramExemplarTTL(time.Minute)

	assert.Equal(t, []float64{0.1, 0.5, 1}, options.Buckets())
	assert.InDelta(t, 1.1, options.NativeHistogramBucketFactor(), 0.0001)
	assert.InDelta(t, 0.001, options.NativeHistogramZeroThreshold(), 0.0001)
	assert.Equal(t, uint32(160), options.NativeHistogramMaxBucketNumber())
	assert.Equal(t, time.Hour, options.NativeHistogramMinResetDuration())
	assert.InDelta(t, 0.01, options.NativeHistogramMaxZeroThreshold(), 0.0001)
	assert.Equal(t, 10, options.NativeHistogramMaxExemplars())
	assert.Equal(t, time.Minute, options.NativeHistogramExemplarTTL())
}

func TestSummaryOptionsDefaults(t *testing.T) {
	t.Parallel()

	options := metrics.NewSummaryOptions()

	assert.Nil(t, options.Objectives())
	assert.Zero(t, options.MaxAge())
	assert.Zero(t, options.AgeBuckets())
	assert.Zero(t, options.BufCap())
}

func TestSummaryOptionsBuilder(t *testing.T) {
	t.Parallel()

	objectives := map[float64]float64{0.5: 0.05, 0.99: 0.001}

	options := metrics.NewSummaryOptions().
		WithObjectives(objectives).
		WithMaxAge(10 * time.Minute).
		WithAgeBuckets(5).
		WithBufCap(500)

	assert.Equal(t, objectives, options.Objectives())
	assert.Equal(t, 10*time.Minute, options.MaxAge())
	assert.Equal(t, uint32(5), options.AgeBuckets())
	assert.Equal(t, uint32(500), options.BufCap())
}
