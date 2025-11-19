package http

import (
	"context"

	"github.com/pixality-inc/golang-core/logger"
)

type RequestMetadataLogsExpander struct {
	log logger.Loggable
}

func NewRequestMetadataLogsExpander() *RequestMetadataLogsExpander {
	return &RequestMetadataLogsExpander{
		log: logger.NewLoggableImplWithService("request_metadata_logs_expander"),
	}
}

func (l *RequestMetadataLogsExpander) Expand(ctx context.Context, logger logger.Logger) logger.Logger {
	requestMetadata, ok := ctx.Value(RequestMetadataValueKey).(*RequestMetadata)
	if !ok {
		return logger
	}

	return logger.WithField("request_id", requestMetadata.RequestId)
}
