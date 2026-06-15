package kafka

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildRecordPlain(t *testing.T) {
	t.Parallel()

	record := buildRecord("events", []byte("payload"), applyProduceOptions())

	assert.Equal(t, "events", record.Topic)
	assert.Equal(t, []byte("payload"), record.Value)
	assert.Nil(t, record.Key)
	assert.Nil(t, record.Headers)
}

func TestBuildRecordWithKey(t *testing.T) {
	t.Parallel()

	record := buildRecord("events", []byte("payload"), applyProduceOptions(WithKey([]byte("entity-1"))))

	assert.Equal(t, []byte("entity-1"), record.Key)
}

func TestBuildRecordWithHeaders(t *testing.T) {
	t.Parallel()

	headers := []Header{
		{Key: "trace-id", Value: []byte("abc")},
		{Key: "source", Value: []byte("test")},
	}

	record := buildRecord("events", []byte("payload"), applyProduceOptions(WithHeaders(headers)))

	require.Len(t, record.Headers, 2)
	assert.Equal(t, "trace-id", record.Headers[0].Key)
	assert.Equal(t, []byte("abc"), record.Headers[0].Value)
	assert.Equal(t, "source", record.Headers[1].Key)
	assert.Equal(t, []byte("test"), record.Headers[1].Value)
}
