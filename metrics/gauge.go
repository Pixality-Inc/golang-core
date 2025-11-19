package metrics

type GaugeImplementation interface {
	Set(value float64)
	Inc()
	Dec()
	Add(value float64)
	Sub(value float64)
	SetToCurrentTime()
}

type Gauge interface {
	Metric[GaugeImplementation]
	GaugeImplementation
}

type GaugeImpl struct {
	*MetricImpl[GaugeImplementation]
	GaugeImplementation
}

func NewGauge(
	description MetricDescription,
	implementation GaugeImplementation,
) *GaugeImpl {
	return &GaugeImpl{
		MetricImpl:          NewMetricImpl(MetricTypeGauge, description, implementation),
		GaugeImplementation: implementation,
	}
}
