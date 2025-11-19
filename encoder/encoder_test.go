package encoder

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEncodeString(t *testing.T) {
	t.Parallel()

	encoder := New([]byte("iddqd"))

	str1 := "Hello"
	str2 := "Привет ❤️"

	encoded1 := encoder.EncodeString(str1)
	require.Equal(t, "IQEIHQs=", encoded1)

	encoded2 := encoder.EncodeString(str2)
	require.Equal(t, "ufu18bTRtNah0bjmRJP5zYvc/g==", encoded2)

	decoded1, err := encoder.DecodeString(encoded1)
	require.NoError(t, err)
	require.Equal(t, str1, decoded1)

	decoded2, err := encoder.DecodeString(encoded2)
	require.NoError(t, err)
	require.Equal(t, str2, decoded2)

	decoded3, err := encoder.DecodeString("IQEIHQs=")
	require.NoError(t, err)
	require.Equal(t, str1, decoded3)

	decoded4, err := encoder.DecodeString("ufu18bTRtNah0bjmRJP5zYvc/g==")
	require.NoError(t, err)
	require.Equal(t, str2, decoded4)
}

func TestEncodeBytes(t *testing.T) {
	t.Parallel()

	encoder := New([]byte("iddqd"))

	str1 := []byte("Hello")
	str2 := []byte("Привет ❤️")

	encoded1 := encoder.Encode(str1)
	require.Equal(t, []byte{0x21, 0x1, 0x8, 0x1d, 0xb}, encoded1)

	encoded2 := encoder.Encode(str2)
	require.Equal(t, []byte{0xb9, 0xfb, 0xb5, 0xf1, 0xb4, 0xd1, 0xb4, 0xd6, 0xa1, 0xd1, 0xb8, 0xe6, 0x44, 0x93, 0xf9, 0xcd, 0x8b, 0xdc, 0xfe}, encoded2)

	decoded1, err := encoder.Decode(encoded1)
	require.NoError(t, err)
	require.Equal(t, str1, decoded1)

	decoded2, err := encoder.Decode(encoded2)
	require.NoError(t, err)
	require.Equal(t, str2, decoded2)

	decoded3, err := encoder.Decode([]byte{0x21, 0x1, 0x8, 0x1d, 0xb})
	require.NoError(t, err)
	require.Equal(t, str1, decoded3)

	decoded4, err := encoder.Decode([]byte{0xb9, 0xfb, 0xb5, 0xf1, 0xb4, 0xd1, 0xb4, 0xd6, 0xa1, 0xd1, 0xb8, 0xe6, 0x44, 0x93, 0xf9, 0xcd, 0x8b, 0xdc, 0xfe})
	require.NoError(t, err)
	require.Equal(t, str2, decoded4)
}

func TestDecodeWrong(t *testing.T) {
	t.Parallel()

	encoder := New([]byte("iddqd"))
	decoder := New([]byte("wrong"))

	str1 := "Hello"
	str2 := "Привет ❤️"

	encoded1 := encoder.EncodeString(str1)
	decoded1, err := decoder.DecodeString(encoded1)
	require.NoError(t, err)
	require.NotEqual(t, str1, decoded1)

	encoded2 := encoder.EncodeString(str2)
	decoded2, err := decoder.DecodeString(encoded2)
	require.NoError(t, err)
	require.NotEqual(t, str2, decoded2)
}

func TestDecodeBadBase64(t *testing.T) {
	t.Parallel()

	encoder := New([]byte("iddqd"))

	decoded, err := encoder.DecodeString("asd")
	require.ErrorIs(t, err, ErrBase64Decode)
	require.Empty(t, decoded)
}
