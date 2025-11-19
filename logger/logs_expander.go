package logger

import (
	"context"
)

type LogsExpandersNameType string

const LogsExpandersName LogsExpandersNameType = "logs_expanders"

type LogsExpanders []LogsExpander

func (l LogsExpanders) Add(logsExpanders ...LogsExpander) LogsExpanders {
	return append(l, logsExpanders...)
}

func GetLogsExpanders(ctx context.Context) LogsExpanders {
	logsExpanders, ok := ctx.Value(LogsExpandersName).(LogsExpanders)
	if !ok {
		logsExpanders = LogsExpanders{}
	}

	return logsExpanders
}

func AddLogsExpanders(ctx context.Context, logsExpanders ...LogsExpander) context.Context {
	return context.WithValue(ctx, LogsExpandersName, append(GetLogsExpanders(ctx), logsExpanders...))
}

func AddLoggerField(ctx context.Context, key string, value any) context.Context {
	return AddLogsExpanders(ctx, NewSingleFieldLogsExpander(key, value))
}

func AddLoggerFields(ctx context.Context, fields Fields) context.Context {
	return AddLogsExpanders(ctx, NewFieldsLogsExpander(fields))
}

type LogsExpander interface {
	Expand(ctx context.Context, logger Logger) Logger
}

type FieldsLogsExpander struct {
	fields Fields
}

func NewSingleFieldLogsExpander(name string, value any) *FieldsLogsExpander {
	return NewFieldsLogsExpander(Fields{
		name: value,
	})
}

func NewFieldsLogsExpander(fields Fields) *FieldsLogsExpander {
	return &FieldsLogsExpander{
		fields: fields,
	}
}

func (f *FieldsLogsExpander) Expand(ctx context.Context, logger Logger) Logger {
	return logger.WithFields(f.fields)
}
