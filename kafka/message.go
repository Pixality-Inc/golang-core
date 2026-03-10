package kafka

import (
	"context"
	"time"

	"github.com/twmb/franz-go/pkg/kgo"
)

type Header struct {
	Key   string
	Value []byte
}

type Message[T any] interface {
	Value() T
	Key() []byte
	Topic() string
	Partition() int32
	Offset() int64
	Timestamp() time.Time
	Headers() []Header
	Commit(ctx context.Context) error
}

type message[T any] struct {
	value     T
	key       []byte
	topic     string
	partition int32
	offset    int64
	timestamp time.Time
	headers   []Header
	commit    func(ctx context.Context) error
}

func (m *message[T]) Value() T             { return m.value }
func (m *message[T]) Key() []byte          { return m.key }
func (m *message[T]) Topic() string        { return m.topic }
func (m *message[T]) Partition() int32     { return m.partition }
func (m *message[T]) Offset() int64        { return m.offset }
func (m *message[T]) Timestamp() time.Time { return m.timestamp }
func (m *message[T]) Headers() []Header    { return m.headers }
func (m *message[T]) Commit(ctx context.Context) error {
	if m.commit == nil {
		return nil
	}

	return m.commit(ctx)
}

func NewMessage[T any](
	value T,
	key []byte,
	topic string,
	partition int32,
	offset int64,
	timestamp time.Time,
	headers []Header,
	commit func(ctx context.Context) error,
) Message[T] {
	return &message[T]{
		value:     value,
		key:       key,
		topic:     topic,
		partition: partition,
		offset:    offset,
		timestamp: timestamp,
		headers:   headers,
		commit:    commit,
	}
}

func convertToKgoHeaders(headers []Header) []kgo.RecordHeader {
	if len(headers) == 0 {
		return nil
	}

	kgoHeaders := make([]kgo.RecordHeader, len(headers))
	for i, h := range headers {
		kgoHeaders[i] = kgo.RecordHeader{
			Key:   h.Key,
			Value: h.Value,
		}
	}

	return kgoHeaders
}

func convertFromKgoHeaders(headers []kgo.RecordHeader) []Header {
	if len(headers) == 0 {
		return nil
	}

	result := make([]Header, len(headers))
	for i, h := range headers {
		result[i] = Header{
			Key:   h.Key,
			Value: h.Value,
		}
	}

	return result
}
