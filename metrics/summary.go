package metrics

// SummaryImplementation for summary metrics
// nolint:iface
type SummaryImplementation interface {
	Observe(value float64)
}

type Summary interface {
	Metric[SummaryImplementation]
	SummaryImplementation

	Options() SummaryOptions
}

type SummaryImpl struct {
	*MetricImpl[SummaryImplementation]
	SummaryImplementation

	options SummaryOptions
}

func NewSummary(
	description MetricDescription,
	options SummaryOptions,
	implementation SummaryImplementation,
) *SummaryImpl {
	return &SummaryImpl{
		MetricImpl:            NewMetricImpl(MetricTypeSummary, description, implementation),
		SummaryImplementation: implementation,
		options:               options,
	}
}

func (m *SummaryImpl) Options() SummaryOptions {
	return m.options
}
