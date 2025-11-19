package metrics

type MetricType string

const (
	MetricTypeCounter   MetricType = "counter"
	MetricTypeGauge     MetricType = "gauge"
	MetricTypeHistogram MetricType = "histogram"
	MetricTypeSummary   MetricType = "summary"
)

type Metric[T any] interface {
	Type() MetricType
	Description() MetricDescription
	Implementation() any
}

type MetricImpl[T any] struct {
	metricType     MetricType
	description    MetricDescription
	implementation T
}

func NewMetricImpl[T any](
	metricType MetricType,
	description MetricDescription,
	implementation T,
) *MetricImpl[T] {
	return &MetricImpl[T]{
		metricType:     metricType,
		description:    description,
		implementation: implementation,
	}
}

func (m *MetricImpl[T]) Type() MetricType {
	return m.metricType
}

func (m *MetricImpl[T]) Description() MetricDescription {
	return m.description
}

func (m *MetricImpl[T]) Implementation() any {
	return m.implementation
}
