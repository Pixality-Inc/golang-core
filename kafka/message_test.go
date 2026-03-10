package kafka

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/twmb/franz-go/pkg/kgo"
)

func TestConvertHeaders_RoundTrip(t *testing.T) {
	t.Parallel()

	headers := []Header{
		{Key: "key1", Value: []byte("value1")},
		{Key: "key2", Value: []byte("value2")},
	}

	kgoHeaders := convertToKgoHeaders(headers)
	require.Len(t, kgoHeaders, 2)
	require.Equal(t, "key1", kgoHeaders[0].Key)
	require.Equal(t, []byte("value1"), kgoHeaders[0].Value)

	back := convertFromKgoHeaders(kgoHeaders)
	require.Equal(t, headers, back)
}

func TestConvertHeaders_Empty(t *testing.T) {
	t.Parallel()

	require.Nil(t, convertToKgoHeaders(nil))
	require.Nil(t, convertToKgoHeaders([]Header{}))
	require.Nil(t, convertFromKgoHeaders(nil))
	require.Nil(t, convertFromKgoHeaders([]kgo.RecordHeader{}))
}

func TestNewMessage(t *testing.T) {
	t.Parallel()

	now := time.Now()
	headers := []Header{{Key: "h1", Value: []byte("v1")}}
	committed := false

	msg := NewMessage(
		"payload",
		[]byte("key"),
		"test-topic",
		int32(2),
		int64(42),
		now,
		headers,
		func(_ context.Context) error {
			committed = true

			return nil
		},
	)

	require.Equal(t, "payload", msg.Value())
	require.Equal(t, []byte("key"), msg.Key())
	require.Equal(t, "test-topic", msg.Topic())
	require.Equal(t, int32(2), msg.Partition())
	require.Equal(t, int64(42), msg.Offset())
	require.Equal(t, now, msg.Timestamp())
	require.Equal(t, headers, msg.Headers())

	require.NoError(t, msg.Commit(context.Background()))
	require.True(t, committed)
}

func TestNewMessage_NilCommit(t *testing.T) {
	t.Parallel()

	msg := NewMessage("value", nil, "topic", 0, 0, time.Time{}, nil, nil)

	require.NoError(t, msg.Commit(context.Background()))
}
