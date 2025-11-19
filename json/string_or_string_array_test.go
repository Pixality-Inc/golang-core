package json

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestStringOrStringArray(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    StringOrStringArray
		wantErr bool
	}{
		{
			name:    "single string",
			input:   `"hello"`,
			want:    StringOrStringArray{"hello"},
			wantErr: false,
		},
		{
			name:    "empty string",
			input:   `""`,
			want:    StringOrStringArray{""},
			wantErr: false,
		},
		{
			name:    "string array with one element",
			input:   `["hello"]`,
			want:    StringOrStringArray{"hello"},
			wantErr: false,
		},
		{
			name:    "string array with multiple elements",
			input:   `["one", "two", "three"]`,
			want:    StringOrStringArray{"one", "two", "three"},
			wantErr: false,
		},
		{
			name:    "empty array",
			input:   `[]`,
			want:    StringOrStringArray{},
			wantErr: false,
		},
		{
			name:    "invalid json",
			input:   `{invalid}`,
			want:    nil,
			wantErr: true,
		},
		{
			name:    "number value",
			input:   `42`,
			want:    nil,
			wantErr: true,
		},
		{
			name:    "boolean value",
			input:   `true`,
			want:    nil,
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()

			var got StringOrStringArray

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

func TestNewStringOrStringArray(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input []string
		want  StringOrStringArray
	}{
		{
			name:  "no arguments",
			input: []string{},
			want:  StringOrStringArray{},
		},
		{
			name:  "single string",
			input: []string{"hello"},
			want:  StringOrStringArray{"hello"},
		},
		{
			name:  "multiple strings",
			input: []string{"one", "two", "three"},
			want:  StringOrStringArray{"one", "two", "three"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := NewStringOrStringArray(tt.input...)
			require.Equal(t, tt.want, got)
		})
	}
}
