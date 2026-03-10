package kafka

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type testMsg struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

func TestJSONProtocol_EncodeDecode(t *testing.T) {
	t.Parallel()

	proto := NewJSONProtocol[testMsg]()
	ctx := context.Background()

	original := testMsg{Name: "test", Count: 42}

	data, err := proto.Encode(ctx, original)
	require.NoError(t, err)
	require.NotEmpty(t, data)

	decoded, err := proto.Decode(ctx, data)
	require.NoError(t, err)
	require.Equal(t, original, decoded)
}

func TestJSONProtocol_DecodeError(t *testing.T) {
	t.Parallel()

	proto := NewJSONProtocol[testMsg]()
	ctx := context.Background()

	_, err := proto.Decode(ctx, []byte("not json"))
	require.Error(t, err)
}

func TestProtobufProtocol_EncodeDecode(t *testing.T) {
	t.Parallel()

	proto := NewProtobufProtocol(func() *wrapperspb.StringValue {
		return &wrapperspb.StringValue{}
	})
	ctx := context.Background()

	original := wrapperspb.String("hello")

	data, err := proto.Encode(ctx, original)
	require.NoError(t, err)
	require.NotEmpty(t, data)

	decoded, err := proto.Decode(ctx, data)
	require.NoError(t, err)
	require.Equal(t, original.GetValue(), decoded.GetValue())
}

func TestProtobufProtocol_DecodeError(t *testing.T) {
	t.Parallel()

	proto := NewProtobufProtocol(func() *wrapperspb.StringValue {
		return &wrapperspb.StringValue{}
	})
	ctx := context.Background()

	// Valid protobuf bytes are hard to make "invalid" since protobuf is lenient,
	// but an incomplete varint will fail.
	_, err := proto.Decode(ctx, []byte{0x80})
	require.Error(t, err)
}
