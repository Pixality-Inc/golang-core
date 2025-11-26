package metrics

import (
	"github.com/pixality-inc/golang-core/clock"
	dto "github.com/prometheus/client_model/go"
)

// Driver implements metrics manager
// nolint:iface
type Driver interface {
	Register(metrics ...Metric[any]) error
	Gather() ([]*dto.MetricFamily, error)
	NewCounter(description MetricDescription) Counter
	NewGauge(description MetricDescription) Gauge
	NewHistogram(description MetricDescription, options HistogramOptions) Histogram
	NewSummary(description MetricDescription, options SummaryOptions) Summary
	NewTimer(clock clock.Clock, observer Observer) Timer
}
