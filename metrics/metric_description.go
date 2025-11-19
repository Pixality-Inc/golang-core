package metrics

import "maps"

type MetricDescription interface {
	Name() string
	Namespace() string
	Subsystem() string
	Help() string
	Labels() map[string]string
}

type MetricDescriptionImpl struct {
	name      string
	namespace string
	subsystem string
	help      string
	labels    map[string]string
}

func NewMetricDescription(name string) *MetricDescriptionImpl {
	return &MetricDescriptionImpl{
		name:      name,
		namespace: "",
		subsystem: "",
		help:      "",
		labels:    make(map[string]string),
	}
}

func (m *MetricDescriptionImpl) WithNamespace(namespace string) *MetricDescriptionImpl {
	m.namespace = namespace

	return m
}

func (m *MetricDescriptionImpl) WithSubsystem(subsystem string) *MetricDescriptionImpl {
	m.subsystem = subsystem

	return m
}

func (m *MetricDescriptionImpl) WithHelp(help string) *MetricDescriptionImpl {
	m.help = help

	return m
}

func (m *MetricDescriptionImpl) WithLabel(key string, value string) *MetricDescriptionImpl {
	m.labels[key] = value

	return m
}

func (m *MetricDescriptionImpl) WithLabels(labels map[string]string) *MetricDescriptionImpl {
	maps.Copy(m.labels, labels)

	return m
}

func (m *MetricDescriptionImpl) Name() string {
	return m.name
}

func (m *MetricDescriptionImpl) Namespace() string {
	return m.namespace
}

func (m *MetricDescriptionImpl) Subsystem() string {
	return m.subsystem
}

func (m *MetricDescriptionImpl) Help() string {
	return m.help
}

func (m *MetricDescriptionImpl) Labels() map[string]string {
	return m.labels
}
