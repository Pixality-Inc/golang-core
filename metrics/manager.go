package metrics

import (
	dto "github.com/prometheus/client_model/go"
)

// Manager implements metrics management
// nolint:iface
type Manager interface {
	Register(metrics ...Metric[any]) error
	Gather() ([]*dto.MetricFamily, error)
	NewCounter(description MetricDescription) Counter
	NewGauge(description MetricDescription) Gauge
	NewHistogram(description MetricDescription, options HistogramOptions) Histogram
	NewSummary(description MetricDescription, options SummaryOptions) Summary
	NewTimer(observer Observer) Timer
}

type Impl struct {
	Driver
}

func New(driver Driver) *Impl {
	return &Impl{
		Driver: driver,
	}
}
