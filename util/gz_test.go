package util

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGzip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   []byte
		wantErr bool
	}{
		{
			name:    "empty data",
			input:   []byte{},
			wantErr: false,
		},
		{
			name:    "simple string",
			input:   []byte("hello world"),
			wantErr: false,
		},
		{
			name:    "long string",
			input:   []byte("this is a longer string that should compress well because it has repeated patterns repeated patterns repeated patterns"),
			wantErr: false,
		},
		{
			name:    "binary data",
			input:   []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE, 0xFD},
			wantErr: false,
		},
	}

	for _, testcase := range tests {
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()

			got, err := Gzip(testcase.input)

			if testcase.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, got)
				// Compressed data should not be empty (gzip has headers)
				if len(testcase.input) > 0 {
					require.NotEmpty(t, got)
				}
			}
		})
	}
}

func TestGunzip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   []byte
		wantErr bool
	}{
		{
			name:    "invalid gzip data",
			input:   []byte("not gzipped"),
			wantErr: true,
		},
		{
			name:    "empty data",
			input:   []byte{},
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()

			got, err := Gunzip(testcase.input)

			if testcase.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, got)
			}
		})
	}
}

func TestGzipGunzipRoundtrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input []byte
	}{
		{
			name:  "empty data",
			input: []byte{},
		},
		{
			name:  "simple string",
			input: []byte("hello world"),
		},
		{
			name:  "long string with repetitions",
			input: []byte("this is a test string " + "repeated pattern " + "repeated pattern " + "repeated pattern"),
		},
		{
			name:  "json data",
			input: []byte(`{"name":"test","value":42,"nested":{"key":"value"}}`),
		},
		{
			name:  "binary data",
			input: []byte{0x00, 0x01, 0x02, 0x03, 0xFF, 0xFE, 0xFD, 0xFC},
		},
		{
			name:  "unicode string",
			input: []byte("Hello мир 世界 🌍"), //nolint:gosmopolitan
		},
		{
			name:  "large data",
			input: []byte(string(make([]byte, 10000))),
		},
	}

	for _, testcase := range tests {
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()

			// Compress
			compressed, err := Gzip(testcase.input)
			require.NoError(t, err)
			require.NotNil(t, compressed)

			// Decompress
			decompressed, err := Gunzip(compressed)
			require.NoError(t, err)
			require.NotNil(t, decompressed)

			// Verify roundtrip
			require.Equal(t, testcase.input, decompressed)
		})
	}
}
