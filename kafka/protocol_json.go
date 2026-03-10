package kafka

import (
	"context"

	"github.com/pixality-inc/golang-core/json"
)

type jsonProtocol[T any] struct{}

func NewJSONProtocol[T any]() Protocol[T] {
	return &jsonProtocol[T]{}
}

func (p *jsonProtocol[T]) DefaultValue() T {
	var zero T

	return zero
}

func (p *jsonProtocol[T]) Encode(_ context.Context, message T) ([]byte, error) {
	return json.Marshal(message)
}

func (p *jsonProtocol[T]) Decode(_ context.Context, data []byte) (T, error) {
	var result T
	if err := json.Unmarshal(data, &result); err != nil {
		return result, err
	}

	return result, nil
}
