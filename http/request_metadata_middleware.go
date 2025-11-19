package http

import (
	"github.com/pixality-inc/golang-core/logger"

	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
)

type RequestMetadata struct {
	Log            logger.Logger
	RequestId      string
	CfIpCountry    string
	CfRay          string
	CfConnectingIp string
}

const RequestMetadataErrorValueKey = "RequestMetadataError"

const RequestMetadataValueKey = "RequestMetadata"

const RequestIdValueKey = "RequestId"

type RequestMetadataMiddleware struct {
	log          logger.Loggable
	logsExpander logger.LogsExpander
}

func NewRequestMetadataMiddleware() *RequestMetadataMiddleware {
	return &RequestMetadataMiddleware{
		log:          logger.NewLoggableImplWithService("request_metadata_middleware"),
		logsExpander: NewRequestMetadataLogsExpander(),
	}
}

func (m *RequestMetadataMiddleware) Handle(next fasthttp.RequestHandler) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		m.handle(ctx, next)
	}
}

func (m *RequestMetadataMiddleware) handle(ctx *fasthttp.RequestCtx, next fasthttp.RequestHandler) {
	requestId := string(ctx.Request.Header.Peek("X-Request-Id"))
	cfIpCountry := string(ctx.Request.Header.Peek("cf-ipcountry"))
	cfRay := string(ctx.Request.Header.Peek("cf-ray"))
	cfConnectingIp := string(ctx.Request.Header.Peek("cf-connecting-ip"))

	if requestId == "" {
		requestId = uuid.New().String()
	}

	requestMetadata := &RequestMetadata{
		Log:            m.log.GetLogger(ctx),
		RequestId:      requestId,
		CfIpCountry:    cfIpCountry,
		CfRay:          cfRay,
		CfConnectingIp: cfConnectingIp,
	}

	ctx.SetUserValue(RequestMetadataValueKey, requestMetadata)

	ctx.SetUserValue(RequestIdValueKey, requestId)

	AddLogsExpanders(ctx, m.logsExpander)

	next(ctx)
}

func GetRequestMetadata(ctx *fasthttp.RequestCtx) *RequestMetadata {
	requestMetadataInterface := ctx.UserValue(RequestMetadataValueKey)

	if requestMetadataInterface == nil {
		return nil
	}

	if requestMetadata, ok := requestMetadataInterface.(*RequestMetadata); !ok {
		return nil
	} else {
		return requestMetadata
	}
}
