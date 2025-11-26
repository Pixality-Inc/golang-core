package metrics

import (
	"github.com/pixality-inc/golang-core/clock"
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

	clock clock.Clock
}

func New(driver Driver, clock clock.Clock) *Impl {
	return &Impl{
		Driver: driver,
		clock:  clock,
	}
}

func (m *Impl) NewTimer(observer Observer) Timer {
	return m.Driver.NewTimer(m.clock, observer)
}
