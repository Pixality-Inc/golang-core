package logger

import (
	"context"
	"maps"
)

//go:generate mockgen -destination mocks/logagable_gen.go -source loggable.go
type Loggable interface {
	GetLoggerWithoutContext() Logger
	GetLogger(ctx context.Context) Logger
}

type LoggableImpl struct {
	logSpawner  LogSpawner
	extraFields Fields
}

func NewLoggableImpl(fields Fields) *LoggableImpl {
	if fields == nil {
		fields = make(Fields)
	}

	return &LoggableImpl{
		logSpawner:  LogSpawnerInstance,
		extraFields: fields,
	}
}

func NewLoggableImplWithService(service string) *LoggableImpl {
	return NewLoggableImpl(Fields{
		"service": service,
	})
}

func NewLoggableImplWithServiceAndFields(service string, fields Fields) *LoggableImpl {
	newFields := Fields{
		"service": service,
	}

	if fields != nil {
		maps.Copy(newFields, fields)
	}

	return NewLoggableImpl(newFields)
}

func (l *LoggableImpl) GetLoggerWithoutContext() Logger {
	return getLoggerWithoutContext(l.logSpawner, l.extraFields)
}

func (l *LoggableImpl) GetLogger(ctx context.Context) Logger {
	return getLogger(ctx, l.logSpawner, l.extraFields)
}
