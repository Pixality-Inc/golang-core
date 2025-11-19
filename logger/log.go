package logger

import (
	"context"
)

func getLoggerWithoutContext(logSpawner LogSpawner, fields Fields) Logger {
	logger := logSpawner.NewLogger().WithFields(fields)

	return logger
}

func getLogger(ctx context.Context, logSpawner LogSpawner, fields Fields) Logger {
	if logSpawner == nil {
		panic("logSpawner is nil")
	}

	logger := logSpawner.NewLogger().WithFields(fields)

	logsExpanders := GetLogsExpanders(ctx)

	for _, logExpander := range logsExpanders {
		logger = logExpander.Expand(ctx, logger)
	}

	return logger
}

func GetLogger(ctx context.Context) Logger {
	return getLogger(ctx, LogSpawnerInstance, nil)
}

func GetLoggerWithoutContext() Logger {
	return getLoggerWithoutContext(LogSpawnerInstance, nil)
}
