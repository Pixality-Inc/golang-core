package kafka

import "context"

type Encoder[T any] interface {
	Encode(ctx context.Context, message T) ([]byte, error)
}

type Decoder[T any] interface {
	DefaultValue() T
	Decode(ctx context.Context, data []byte) (T, error)
}

type Protocol[T any] interface {
	Encoder[T]
	Decoder[T]
}
