package kafka

import "github.com/twmb/franz-go/pkg/kgo"

type produceConfig struct {
	Key     []byte
	Headers []Header
}

type ProduceOption func(*produceConfig)

func WithKey(key []byte) ProduceOption {
	return func(cfg *produceConfig) {
		cfg.Key = key
	}
}

func WithHeaders(headers []Header) ProduceOption {
	return func(cfg *produceConfig) {
		cfg.Headers = headers
	}
}

func applyProduceOptions(opts ...ProduceOption) *produceConfig {
	cfg := &produceConfig{}
	for _, opt := range opts {
		opt(cfg)
	}

	return cfg
}

func buildRecord(topic string, data []byte, cfg *produceConfig) *kgo.Record {
	record := &kgo.Record{
		Topic: topic,
		Value: data,
	}

	if cfg.Key != nil {
		record.Key = cfg.Key
	}

	if cfg.Headers != nil {
		record.Headers = convertToKgoHeaders(cfg.Headers)
	}

	return record
}
