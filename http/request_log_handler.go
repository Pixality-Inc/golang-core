package http

import (
	"strings"

	"github.com/pixality-inc/golang-core/logger"
	"github.com/pixality-inc/golang-core/timetrack"

	realip "github.com/ferluci/fast-realip"
	"github.com/valyala/fasthttp"
)

func RequestLogHandler(originalHandler fasthttp.RequestHandler) fasthttp.RequestHandler {
	loggable := logger.NewLoggableImplWithService("request_log_handler")

	return func(ctx *fasthttp.RequestCtx) {
		requestTimeTracker := timetrack.New(ctx)

		originalHandler(ctx)

		requestTimeTracker.Finish()

		// Request Metadata

		log := loggable.GetLogger(ctx)

		// Log

		uri := ctx.Request.URI()

		if strings.HasPrefix(string(uri.Path()), "/healthcheck") {
			return
		}

		statusCode := ctx.Response.StatusCode()

		clientIP := realip.FromRequest(ctx)

		requestMetadataError, ok := ctx.UserValue(RequestMetadataErrorValueKey).(error)
		if !ok {
			requestMetadataError = nil
		}

		logEntry := log.
			WithField("success", statusCode >= 200 && statusCode <= 299).
			WithField("http_method", string(ctx.Request.Header.Method())).
			WithField("http_user_agent", string(ctx.Request.Header.UserAgent())).
			WithField("client_ip", clientIP).
			WithField("status_code", statusCode).
			WithField("response_time", requestTimeTracker.Duration().Milliseconds()).
			WithField("response_bytes", len(ctx.Response.Body())).
			WithField("logger", "router")

		if requestMetadataError != nil {
			logEntry = logEntry.WithError(requestMetadataError)
		}

		requestMetadata := GetRequestMetadata(ctx)

		if requestMetadata != nil {
			if requestMetadata.RequestId != "" {
				logEntry = logEntry.WithField("request_id", requestMetadata.RequestId)
			}

			if requestMetadata.CfIpCountry != "" {
				logEntry = logEntry.WithField("cf_ip_country", requestMetadata.CfIpCountry)
			}

			if requestMetadata.CfRay != "" {
				logEntry = logEntry.WithField("cf_ray", requestMetadata.CfRay)
			}

			if requestMetadata.CfConnectingIp != "" {
				logEntry = logEntry.WithField("cf_connecting_ip", requestMetadata.CfConnectingIp)
			}
		}

		logEntry.Debug(uri.String())

		if requestMetadataError != nil {
			logEntry.
				WithField("uri", uri.String()).
				Error("Request failed")
		}
	}
}
