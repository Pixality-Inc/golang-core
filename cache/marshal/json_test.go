package marshal_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pixality-inc/golang-core/cache/marshal"
)

func TestJsonMarshaller_Marshal(t *testing.T) {
	t.Parallel()

	type payload struct {
		A int    `json:"a"`
		B string `json:"b"`
	}

	tests := []struct {
		name    string
		input   any
		want    string
		wantErr bool
	}{
		{
			name:  "struct",
			input: payload{A: 1, B: "x"},
			want:  `{"a":1,"b":"x"}`,
		},
		{
			name:  "map",
			input: map[string]any{"a": 1},
			want:  `{"a":1}`,
		},
		{
			name:    "unsupported type",
			input:   func() {},
			wantErr: true,
		},
	}

	marshaller := marshal.NewJsonMarshaller()

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			data, err := marshaller.Marshal(testCase.input)

			if testCase.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			require.JSONEq(t, testCase.want, string(data))
		})
	}
}

func TestJsonMarshaller_Unmarshal(t *testing.T) {
	t.Parallel()

	type payload struct {
		A int    `json:"a"`
		B string `json:"b"`
	}

	tests := []struct {
		name    string
		input   []byte
		result  any
		assert  func(t *testing.T, result any)
		wantErr bool
	}{
		{
			name:   "struct_pointer",
			input:  []byte(`{"a":10,"b":"hello"}`),
			result: &payload{},
			assert: func(t *testing.T, r any) {
				t.Helper()

				p := r.(*payload) // nolint:errcheck,forcetypeassert
				require.Equal(t, 10, p.A)
				require.Equal(t, "hello", p.B)
			},
		},
		{
			name:   "map_pointer",
			input:  []byte(`{"a":1}`),
			result: &map[string]any{},
			assert: func(t *testing.T, r any) {
				t.Helper()

				m := r.(*map[string]any) // nolint:errcheck,forcetypeassert
				require.InEpsilon(t, float64(1), (*m)["a"], 0)
			},
		},
		{
			name:    "invalid_json",
			input:   []byte(`{invalid}`),
			result:  &payload{},
			wantErr: true,
		},
	}

	marshaller := marshal.NewJsonMarshaller()

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			err := marshaller.Unmarshal(testCase.input, testCase.result)

			if testCase.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			require.NotNil(t, testCase.assert)
			testCase.assert(t, testCase.result)
		})
	}
}
