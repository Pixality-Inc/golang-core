package http

import (
	"errors"

	"github.com/pixality-inc/golang-core/logger"

	"github.com/valyala/fasthttp"
	"google.golang.org/protobuf/proto"
)

type ProtocolRenderer interface {
	Ok() proto.Message
	Error(statusCode int, err error) proto.Message
}

type ResponseRenderer interface {
	EmptyOk(ctx *fasthttp.RequestCtx)
	Ok(ctx *fasthttp.RequestCtx, response proto.Message)
	Created(ctx *fasthttp.RequestCtx, response proto.Message)
	Error(ctx *fasthttp.RequestCtx, err error)
	InternalServerError(ctx *fasthttp.RequestCtx, err error)
	BadRequest(ctx *fasthttp.RequestCtx, err error)
	NotFound(ctx *fasthttp.RequestCtx, err error)
	Unauthorized(ctx *fasthttp.RequestCtx, err error)
	Forbidden(ctx *fasthttp.RequestCtx, err error)
}

type ResponseRendererImpl struct {
	log           logger.Loggable
	protoRenderer ProtocolRenderer
}

func NewResponseRenderer(protoRenderer ProtocolRenderer) *ResponseRendererImpl {
	return &ResponseRendererImpl{
		log:           logger.NewLoggableImplWithService("response_renderer"),
		protoRenderer: protoRenderer,
	}
}

func (r *ResponseRendererImpl) EmptyOk(ctx *fasthttp.RequestCtx) {
	r.Ok(ctx, r.protoRenderer.Ok())
}

func (r *ResponseRendererImpl) Ok(ctx *fasthttp.RequestCtx, response proto.Message) {
	if err := renderResponse(ctx, fasthttp.StatusOK, response); err != nil {
		r.log.GetLogger(ctx).WithError(err).Error("output error")
	}
}

func (r *ResponseRendererImpl) Created(ctx *fasthttp.RequestCtx, response proto.Message) {
	if err := renderResponse(ctx, fasthttp.StatusCreated, response); err != nil {
		r.log.GetLogger(ctx).WithError(err).Error("output error")
	}
}

func (r *ResponseRendererImpl) Error(ctx *fasthttp.RequestCtx, err error) {
	if err != nil {
		ctx.SetUserValue(RequestMetadataErrorValueKey, err)
	}

	var statusCode int

	switch {
	case errors.Is(err, ErrBadRequest):
		statusCode = fasthttp.StatusBadRequest

	case errors.Is(err, ErrNotFound):
		statusCode = fasthttp.StatusNotFound

	case errors.Is(err, ErrUnauthorized):
		statusCode = fasthttp.StatusUnauthorized

	case errors.Is(err, ErrForbidden):
		statusCode = fasthttp.StatusForbidden

	case errors.Is(err, ErrInternalServerError):
		statusCode = fasthttp.StatusInternalServerError

	default:
		statusCode = fasthttp.StatusInternalServerError
	}

	errorMessage := r.protoRenderer.Error(statusCode, err)

	if err := renderResponse(ctx, statusCode, errorMessage); err != nil {
		r.log.GetLogger(ctx).WithError(err).Error("output error")
	}
}

func (r *ResponseRendererImpl) InternalServerError(ctx *fasthttp.RequestCtx, err error) {
	r.Error(ctx, errors.Join(ErrInternalServerError, err))
}

func (r *ResponseRendererImpl) BadRequest(ctx *fasthttp.RequestCtx, err error) {
	r.Error(ctx, errors.Join(ErrBadRequest, err))
}

func (r *ResponseRendererImpl) NotFound(ctx *fasthttp.RequestCtx, err error) {
	r.Error(ctx, errors.Join(ErrNotFound, err))
}

func (r *ResponseRendererImpl) Unauthorized(ctx *fasthttp.RequestCtx, err error) {
	r.Error(ctx, errors.Join(ErrUnauthorized, err))
}

func (r *ResponseRendererImpl) Forbidden(ctx *fasthttp.RequestCtx, err error) {
	r.Error(ctx, errors.Join(ErrForbidden, err))
}
