package drivers

import (
	"errors"
	"fmt"

	"github.com/pixality-inc/golang-core/clock"
	"github.com/pixality-inc/golang-core/metrics"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	dto "github.com/prometheus/client_model/go"
)

var ErrMetricIsNotAValidCollector = errors.New("metric is not a valid collector")

type PrometheusDriver struct {
	registry *prometheus.Registry
}

func NewPrometheusDriver(
	withGoCollector bool,
	withNewProcessCollector bool,
) *PrometheusDriver {
	registry := prometheus.NewRegistry()

	if withGoCollector {
		registry.MustRegister(collectors.NewGoCollector())
	}

	if withNewProcessCollector {
		registry.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	}

	return &PrometheusDriver{
		registry: registry,
	}
}

func (d *PrometheusDriver) Register(metrics ...metrics.Metric[any]) error {
	for _, metric := range metrics {
		metricDescription := metric.Description()

		collector, ok := metric.Implementation().(prometheus.Collector)
		if !ok {
			return fmt.Errorf("failed to register prometheus metric %s (%s): %w", metricDescription.Name(), metric.Type(), ErrMetricIsNotAValidCollector)
		}

		if err := d.registry.Register(collector); err != nil {
			return fmt.Errorf("register prometheus collector for %s (%s): %w", metricDescription.Name(), metric.Type(), err)
		}
	}

	return nil
}

func (d *PrometheusDriver) Gather() ([]*dto.MetricFamily, error) {
	return d.registry.Gather()
}

func (d *PrometheusDriver) NewCounter(
	description metrics.MetricDescription,
) metrics.Counter {
	return metrics.NewCounter(
		description,
		prometheus.NewCounter(prometheus.CounterOpts(d.getMetricsOpts(description))),
	)
}

func (d *PrometheusDriver) NewGauge(
	description metrics.MetricDescription,
) metrics.Gauge {
	return metrics.NewGauge(
		description,
		prometheus.NewGauge(prometheus.GaugeOpts(d.getMetricsOpts(description))),
	)
}

func (d *PrometheusDriver) NewHistogram(
	description metrics.MetricDescription,
	options metrics.HistogramOptions,
) metrics.Histogram {
	opts := d.getMetricsOpts(description)

	histogram := prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace:                       opts.Namespace,
		Subsystem:                       opts.Subsystem,
		Name:                            opts.Name,
		Help:                            opts.Help,
		ConstLabels:                     opts.ConstLabels,
		Buckets:                         options.Buckets(),
		NativeHistogramBucketFactor:     options.NativeHistogramBucketFactor(),
		NativeHistogramZeroThreshold:    options.NativeHistogramZeroThreshold(),
		NativeHistogramMaxBucketNumber:  options.NativeHistogramMaxBucketNumber(),
		NativeHistogramMinResetDuration: options.NativeHistogramMinResetDuration(),
		NativeHistogramMaxZeroThreshold: options.NativeHistogramMaxZeroThreshold(),
		NativeHistogramMaxExemplars:     options.NativeHistogramMaxExemplars(),
		NativeHistogramExemplarTTL:      options.NativeHistogramExemplarTTL(),
	})

	return metrics.NewHistogram(description, options, histogram)
}

func (d *PrometheusDriver) NewSummary(
	description metrics.MetricDescription,
	options metrics.SummaryOptions,
) metrics.Summary {
	opts := d.getMetricsOpts(description)

	summary := prometheus.NewSummary(prometheus.SummaryOpts{
		Namespace:   opts.Namespace,
		Subsystem:   opts.Subsystem,
		Name:        opts.Name,
		Help:        opts.Help,
		ConstLabels: opts.ConstLabels,
		Objectives:  options.Objectives(),
		MaxAge:      options.MaxAge(),
		AgeBuckets:  options.AgeBuckets(),
		BufCap:      options.BufCap(),
	})

	return metrics.NewSummary(description, options, summary)
}

func (d *PrometheusDriver) NewTimer(
	observer metrics.Observer,
) metrics.Timer {
	return metrics.NewTimer(clock.New(), observer)
}

func (d *PrometheusDriver) getMetricsOpts(metricDescription metrics.MetricDescription) prometheus.Opts {
	opts := prometheus.Opts{
		Name:        metricDescription.Name(),
		Namespace:   metricDescription.Namespace(),
		Subsystem:   metricDescription.Subsystem(),
		Help:        metricDescription.Help(),
		ConstLabels: nil,
	}

	labels := metricDescription.Labels()

	if len(labels) > 0 {
		opts.ConstLabels = labels
	}

	return opts
}
