package http

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/pixality-inc/golang-core/logger"

	"github.com/pkg/errors"
	"github.com/valyala/fasthttp"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var (
	ErrNoResponseRenderer            = errors.New("no response renderer")
	ErrNoHttpResponseModelAndOptions = errors.New("no http response model and options")
)

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

func HandleHttp[T proto.Message](ctx *fasthttp.RequestCtx, response HttpResponse[T]) {
	model := response.Model()
	options := response.Options()

	if model == nil && len(options) == 0 {
		InternalServerError(ctx, ErrNoHttpResponseModelAndOptions)

		return
	}

	if model != nil {
		Ok(ctx, *model)
	}

	for _, option := range options {
		option(ctx)
	}
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

// acceptRange represents a single media-range from Accept header with its quality value
type acceptRange struct {
	mediaType string
	q         float64
}

// parseAcceptHeader parses Accept/Content-Type header value according to RFC 7231
// example: "*/*; q=0.5, application/xml" -> [{*/* 0.5}, {application/xml 1.0}]
func parseAcceptHeader(header string) []acceptRange {
	if header == "" {
		return []acceptRange{{mediaType: "", q: 1.0}}
	}

	parts := strings.Split(header, ",")
	result := make([]acceptRange, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		mediaType := part
		quality := 1.0

		if mtype, params, ok := strings.Cut(part, ";"); ok {
			mediaType = strings.TrimSpace(mtype)
			// parse params looking for q
			for param := range strings.SplitSeq(params, ";") {
				param = strings.TrimSpace(param)
				if strings.HasPrefix(param, "q=") {
					if v, err := strconv.ParseFloat(strings.TrimSpace(param[2:]), 64); err == nil && v >= 0 && v <= 1 {
						quality = v
					}

					break
				}
			}
		}

		result = append(result, acceptRange{mediaType: mediaType, q: quality})
	}

	return result
}

// mediaTypeToFormat maps a media type string to internal dataFormatType.
// returns DataFormatUnknown for unsupported types.
func mediaTypeToFormat(mediaType string) dataFormatType {
	switch mediaType {
	case "application/json":
		return DataFormatJson
	case "application/protobuf":
		return DataFormatProtobuf
	case "application/x-protobuf":
		return DataFormatXProtobuf
	case "*/*":
		return DataFormatJson
	case "":
		return DataFormatJson
	default:
		return DataFormatUnknown
	}
}

func getAcceptFormat(ctx *fasthttp.RequestCtx, headerName string) (dataFormatType, error) {
	raw := strings.ToLower(strings.TrimSpace(string(ctx.Request.Header.Peek(headerName))))

	ranges := parseAcceptHeader(raw)
	sort.SliceStable(ranges, func(i, j int) bool {
		return ranges[i].q > ranges[j].q
	})

	for _, r := range ranges {
		if r.q == 0 {
			continue
		}
		if f := mediaTypeToFormat(r.mediaType); f != DataFormatUnknown {
			return f, nil
		}
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
