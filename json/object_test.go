package json

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAsJsonObject(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   RawMessage
		want    Object
		wantErr bool
	}{
		{
			name:  "simple object",
			input: RawMessage(`{"key":"value"}`),
			want: Object{
				"key": "value",
			},
			wantErr: false,
		},
		{
			name:  "nested object",
			input: RawMessage(`{"outer":{"inner":"value"}}`),
			want: Object{
				"outer": map[string]any{
					"inner": "value",
				},
			},
			wantErr: false,
		},
		{
			name:    "empty object",
			input:   RawMessage(`{}`),
			want:    Object{},
			wantErr: false,
		},
		{
			name:  "object with multiple types",
			input: RawMessage(`{"string":"text","number":42,"bool":true,"null":null}`),
			want: Object{
				"string": "text",
				"number": float64(42),
				"bool":   true,
				"null":   nil,
			},
			wantErr: false,
		},
		{
			name:    "invalid json",
			input:   RawMessage(`{invalid}`),
			want:    nil,
			wantErr: true,
		},
		{
			name:    "array instead of object",
			input:   RawMessage(`["not","an","object"]`),
			want:    nil,
			wantErr: true,
		},
	}

	for _, testcase := range tests {
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()

			got, err := AsJsonObject(testcase.input)

			if testcase.wantErr {
				require.Error(t, err)
				require.Nil(t, got)
			} else {
				require.NoError(t, err)
				require.Equal(t, testcase.want, got)
			}
		})
	}
}
