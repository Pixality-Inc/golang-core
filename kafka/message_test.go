package kafka

import (
	"testing"

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
