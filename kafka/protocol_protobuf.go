package kafka

import (
	"context"

	"google.golang.org/protobuf/proto"
)

type protobufProtocol[T proto.Message] struct {
	factory func() T
}

func NewProtobufProtocol[T proto.Message](factory func() T) Protocol[T] {
	return &protobufProtocol[T]{
		factory: factory,
	}
}

func (p *protobufProtocol[T]) DefaultValue() T {
	return p.factory()
}

func (p *protobufProtocol[T]) Encode(_ context.Context, message T) ([]byte, error) {
	return proto.Marshal(message)
}

func (p *protobufProtocol[T]) Decode(_ context.Context, data []byte) (T, error) {
	msg := p.factory()
	if err := proto.Unmarshal(data, msg); err != nil {
		var zero T

		return zero, err
	}

	return msg, nil
}
