package http_client

import (
	"fmt"

	"github.com/pixality-inc/golang-core/json"
)

//go:generate mockgen -destination mocks/response_gen.go -source response.go
type Response interface {
	GetStatusCode() int
	GetHeaders() Headers
	GetBody() []byte
	DecodeJSON(v any) error
	String() string
}

type TypedResponse[OUT any] interface {
	GetStatusCode() int
	GetHeaders() Headers
	GetBody() []byte
	GetEntity() OUT
	String() string
}

type ResponseImpl struct {
	StatusCode int
	Headers    Headers
	Body       []byte
}

func (r *ResponseImpl) GetStatusCode() int {
	return r.StatusCode
}

func (r *ResponseImpl) GetHeaders() Headers {
	return r.Headers
}

func (r *ResponseImpl) GetBody() []byte {
	return r.Body
}

func (r *ResponseImpl) DecodeJSON(v any) error {
	return json.Unmarshal(r.Body, v)
}

func (r *ResponseImpl) String() string {
	return fmt.Sprintf("StatusCode: %d, Body: %s", r.StatusCode, string(r.Body))
}

type TypedResponseImpl[OUT any] struct {
	StatusCode int
	Headers    Headers
	Body       []byte
	Entity     OUT
}

func (r *TypedResponseImpl[OUT]) GetStatusCode() int {
	return r.StatusCode
}

func (r *TypedResponseImpl[OUT]) GetHeaders() Headers {
	return r.Headers
}

func (r *TypedResponseImpl[OUT]) GetBody() []byte {
	return r.Body
}

func (r *TypedResponseImpl[OUT]) GetEntity() OUT {
	return r.Entity
}

func (r *TypedResponseImpl[OUT]) String() string {
	return fmt.Sprintf("StatusCode: %d, Entity: %+v", r.StatusCode, r.Entity)
}
