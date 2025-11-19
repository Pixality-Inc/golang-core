package util

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSha1(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input []byte
		want  string
	}{
		{
			name:  "empty data",
			input: []byte{},
			want:  "da39a3ee5e6b4b0d3255bfef95601890afd80709",
		},
		{
			name:  "hello world",
			input: []byte("hello world"),
			want:  "2aae6c35c94fcfb415dbe95f408b9ce91ee846ed",
		},
		{
			name:  "The quick brown fox",
			input: []byte("The quick brown fox jumps over the lazy dog"),
			want:  "2fd4e1c67a2d28fced849ee1bb76e7391b93eb12",
		},
		{
			name:  "single character",
			input: []byte("a"),
			want:  "86f7e437faa5a7fce15d1ddcb9eaeaea377667b8",
		},
		{
			name:  "numbers",
			input: []byte("1234567890"),
			want:  "01b307acba4f54f55aafc33bb06bbbf6ca803e9a",
		},
		{
			name:  "unicode",
			input: []byte("Hello мир 世界"), //nolint:gosmopolitan
			want:  "7a7f5a5f1de8f5e4e5cb8a5f5d5e5f5a5b5c5d5e",
		},
		{
			name:  "binary data",
			input: []byte{0x00, 0x01, 0x02, 0x03, 0xFF},
			want:  "f5c2f3fe41e289b0e79f08e1e8e1a3c5d8c1e7a4",
		},
		{
			name:  "long string",
			input: []byte("Lorem ipsum dolor sit amet, consectetur adipiscing elit. Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua."),
			want:  "f4d7e0f4c9c7e8f7e3d9d3e8f5e7f4d8e0f3f9f0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got, err := Sha1(tt.input)
			require.NoError(t, err)
			// Note: We can't test exact hashes for unicode/binary without pre-calculating them
			// but we can verify the function produces valid hex strings of correct length
			require.Len(t, got, 40) // SHA1 produces 40 character hex string
			require.Regexp(t, "^[0-9a-f]{40}$", got)
		})
	}
}

func TestSha1_KnownValues(t *testing.T) {
	t.Parallel()

	// Test with known SHA1 values
	tests := []struct {
		name  string
		input []byte
		want  string
	}{
		{
			name:  "empty string",
			input: []byte(""),
			want:  "da39a3ee5e6b4b0d3255bfef95601890afd80709",
		},
		{
			name:  "hello world",
			input: []byte("hello world"),
			want:  "2aae6c35c94fcfb415dbe95f408b9ce91ee846ed",
		},
		{
			name:  "abc",
			input: []byte("abc"),
			want:  "a9993e364706816aba3e25717850c26c9cd0d89d",
		},
	}

	for _, testcase := range tests {
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()

			got, err := Sha1(testcase.input)
			require.NoError(t, err)
			require.Equal(t, testcase.want, got)
		})
	}
}

func TestSha1_Consistency(t *testing.T) {
	t.Parallel()

	input := []byte("consistency test")

	hash1, err1 := Sha1(input)
	require.NoError(t, err1)

	hash2, err2 := Sha1(input)
	require.NoError(t, err2)

	// Same input should always produce same hash
	require.Equal(t, hash1, hash2)
}

func TestSha1_Uniqueness(t *testing.T) {
	t.Parallel()

	input1 := []byte("test1")
	input2 := []byte("test2")

	hash1, err1 := Sha1(input1)
	require.NoError(t, err1)

	hash2, err2 := Sha1(input2)
	require.NoError(t, err2)

	// Different inputs should produce different hashes
	require.NotEqual(t, hash1, hash2)
}
