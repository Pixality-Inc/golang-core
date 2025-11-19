package json

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStringified(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    Stringified
		wantErr bool
	}{
		{
			name:    "string value",
			input:   `"hello"`,
			want:    Stringified("hello"),
			wantErr: false,
		},
		{
			name:    "integer value",
			input:   `42`,
			want:    Stringified("42"),
			wantErr: false,
		},
		{
			name:    "negative integer",
			input:   `-123`,
			want:    Stringified("-123"),
			wantErr: false,
		},
		{
			name:    "zero",
			input:   `0`,
			want:    Stringified("0"),
			wantErr: false,
		},
		{
			name:    "empty string",
			input:   `""`,
			want:    Stringified(""),
			wantErr: false,
		},
		{
			name:    "invalid json",
			input:   `{invalid}`,
			want:    Stringified(""),
			wantErr: true,
		},
		{
			name:    "boolean value",
			input:   `true`,
			want:    Stringified(""),
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()

			var got Stringified

			err := Unmarshal([]byte(testcase.input), &got)

			if testcase.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.Equal(t, testcase.want, got)
			}
		})
	}
}
