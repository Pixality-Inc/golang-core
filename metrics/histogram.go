package metrics

// HistogramImplementation for histogram metrics
// nolint:iface
type HistogramImplementation interface {
	Observe(value float64)
}

type Histogram interface {
	Metric[HistogramImplementation]
	HistogramImplementation

	Options() HistogramOptions
}

type HistogramImpl struct {
	*MetricImpl[HistogramImplementation]
	HistogramImplementation

	options HistogramOptions
}

func NewHistogram(
	description MetricDescription,
	options HistogramOptions,
	implementation HistogramImplementation,
) *HistogramImpl {
	return &HistogramImpl{
		MetricImpl:              NewMetricImpl(MetricTypeHistogram, description, implementation),
		HistogramImplementation: implementation,
		options:                 options,
	}
}

func (m *HistogramImpl) Options() HistogramOptions {
	return m.options
}
