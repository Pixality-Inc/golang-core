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

type Message[T any] struct {
	Value     T
	Key       []byte
	Topic     string
	Partition int32
	Offset    int64
	Timestamp time.Time
	Headers   []Header
	Commit    func(ctx context.Context) error
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
