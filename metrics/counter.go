package metrics

type CounterImplementation interface {
	Inc()
	Add(value float64)
}

type Counter interface {
	Metric[CounterImplementation]
	CounterImplementation
}

type CounterImpl struct {
	*MetricImpl[CounterImplementation]
	CounterImplementation
}

func NewCounter(
	description MetricDescription,
	implementation CounterImplementation,
) *CounterImpl {
	return &CounterImpl{
		MetricImpl:            NewMetricImpl[CounterImplementation](MetricTypeCounter, description, implementation),
		CounterImplementation: implementation,
	}
}
