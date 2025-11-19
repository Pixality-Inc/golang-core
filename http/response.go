package http

import (
	"fmt"
	"strings"

	"github.com/pixality-inc/golang-core/logger"

	"github.com/pkg/errors"
	"github.com/valyala/fasthttp"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var ErrNoResponseRenderer = errors.New("no response renderer")

type dataFormatType int

const (
	DataFormatUnknown   = 0
	DataFormatJson      = 1
	DataFormatProtobuf  = 2
	DataFormatXProtobuf = 3
)

func ReadBody(ctx *fasthttp.RequestCtx, obj proto.Message) error {
	format, err := getInputFormat(ctx)
	if err != nil {
		return err
	}

	bytes := ctx.Request.Body()

	switch format {
	case DataFormatJson:
		err = jsonUnmarshaller.Unmarshal(bytes, obj)
		// err = json.Unmarshal(bytes, obj)

	case DataFormatProtobuf:
		err = proto.Unmarshal(bytes, obj)

	case DataFormatXProtobuf:
		err = proto.Unmarshal(bytes, obj)
	}

	return err
}

func EmptyOk(ctx *fasthttp.RequestCtx) {
	withResponseRenderer(ctx, func(rr ResponseRenderer) {
		rr.EmptyOk(ctx)
	})
}

func Ok(ctx *fasthttp.RequestCtx, response proto.Message) {
	withResponseRenderer(ctx, func(rr ResponseRenderer) {
		rr.Ok(ctx, response)
	})
}

func Created(ctx *fasthttp.RequestCtx, response proto.Message) {
	withResponseRenderer(ctx, func(rr ResponseRenderer) {
		rr.Created(ctx, response)
	})
}

func HandleError(ctx *fasthttp.RequestCtx, err error) {
	Error(ctx, err)
}

func Error(ctx *fasthttp.RequestCtx, err error) {
	withResponseRenderer(ctx, func(rr ResponseRenderer) {
		rr.Error(ctx, err)
	})
}

func InternalServerError(ctx *fasthttp.RequestCtx, errs ...error) {
	withResponseRenderer(ctx, func(rr ResponseRenderer) {
		if len(errs) > 0 {
			rr.InternalServerError(ctx, errs[0])
		} else {
			rr.InternalServerError(ctx, nil)
		}
	})
}

func BadRequest(ctx *fasthttp.RequestCtx, errs ...error) {
	withResponseRenderer(ctx, func(rr ResponseRenderer) {
		if len(errs) > 0 {
			rr.BadRequest(ctx, errs[0])
		} else {
			rr.BadRequest(ctx, nil)
		}
	})
}

func NotFound(ctx *fasthttp.RequestCtx, errs ...error) {
	withResponseRenderer(ctx, func(rr ResponseRenderer) {
		if len(errs) > 0 {
			rr.NotFound(ctx, errs[0])
		} else {
			rr.NotFound(ctx, nil)
		}
	})
}

func Unauthorized(ctx *fasthttp.RequestCtx, errs ...error) {
	withResponseRenderer(ctx, func(rr ResponseRenderer) {
		if len(errs) > 0 {
			rr.Unauthorized(ctx, errs[0])
		} else {
			rr.Unauthorized(ctx, nil)
		}
	})
}

func Forbidden(ctx *fasthttp.RequestCtx, errs ...error) {
	withResponseRenderer(ctx, func(rr ResponseRenderer) {
		if len(errs) > 0 {
			rr.Forbidden(ctx, errs[0])
		} else {
			rr.Forbidden(ctx, nil)
		}
	})
}

func renderResponse(ctx *fasthttp.RequestCtx, statusCode int, response proto.Message) error {
	format, err := getOutputFormat(ctx)
	if err != nil {
		return fmt.Errorf("getting output format: %w", err)
	}

	var responseBytes []byte

	var contentType string

	switch format {
	case DataFormatJson:
		responseBytes, err = jsonMarshaller.Marshal(response)
		contentType = "application/json"

	case DataFormatProtobuf:
		responseBytes, err = proto.Marshal(response)
		contentType = "application/protobuf"

	case DataFormatXProtobuf:
		responseBytes, err = proto.Marshal(response)
		contentType = "application/x-protobuf"
	}

	if err != nil {
		return err
	}

	ctx.SetStatusCode(statusCode)
	ctx.Response.Header.Set("Content-Type", contentType)
	ctx.SetBody(responseBytes)

	return nil
}

func withResponseRenderer(ctx *fasthttp.RequestCtx, fn func(responseRenderer ResponseRenderer)) {
	responseRenderer := GetResponseRenderer(ctx)
	if responseRenderer != nil {
		fn(responseRenderer)
	} else {
		logger.GetLogger(ctx).WithError(ErrNoResponseRenderer).Error("no response renderer")
	}
}

func getInputFormat(ctx *fasthttp.RequestCtx) (dataFormatType, error) {
	return getAcceptFormat(ctx, "Content-Type")
}

func getOutputFormat(ctx *fasthttp.RequestCtx) (dataFormatType, error) {
	return getAcceptFormat(ctx, "Accept")
}

func getAcceptFormat(ctx *fasthttp.RequestCtx, headerName string) (dataFormatType, error) {
	acceptHeaderStr := strings.ToLower(string(ctx.Request.Header.Peek(headerName)))

	switch acceptHeaderStr {
	case "application/json":
		return DataFormatJson, nil
	case "application/protobuf":
		return DataFormatProtobuf, nil
	case "application/x-protobuf":
		return DataFormatXProtobuf, nil
	case "*/*":
		return DataFormatProtobuf, nil
	case "":
		return DataFormatProtobuf, nil
	}

	return DataFormatUnknown, errors.New(fmt.Sprintf("can't recognize data format (in header %v)", headerName))
}

var jsonMarshaller = protojson.MarshalOptions{
	UseProtoNames:   true,
	Multiline:       false,
	EmitUnpopulated: true,
	UseEnumNumbers:  true,
}

var jsonUnmarshaller = protojson.UnmarshalOptions{
	AllowPartial:   false,
	DiscardUnknown: false,
}
