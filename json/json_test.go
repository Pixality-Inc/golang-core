package json

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/require"
)

type marshalInner struct {
	Flag  *bool   `json:"flag,omitempty"`
	Label *string `json:"label,omitempty"`
}

type marshalOuter struct {
	Inner *marshalInner `json:"inner,omitempty"`
}

// the shape that segfaulted the goccy encoder: an omitempty pointer to a struct of pointers. the
// failure was an unrecoverable fatal error, so a regression here kills the test binary rather than
// failing the assertion
func TestMarshalPointerToStructOfPointers(t *testing.T) {
	t.Parallel()

	flag := false
	label := "value"

	tests := []struct {
		name  string
		input marshalOuter
		want  string
	}{
		{
			name:  "absent inner",
			input: marshalOuter{},
			want:  `{}`,
		},
		{
			name:  "present but empty inner",
			input: marshalOuter{Inner: &marshalInner{}},
			want:  `{"inner":{}}`,
		},
		{
			name:  "inner holding a bool",
			input: marshalOuter{Inner: &marshalInner{Flag: &flag}},
			want:  `{"inner":{"flag":false}}`,
		},
		{
			name:  "inner holding every field",
			input: marshalOuter{Inner: &marshalInner{Flag: &flag, Label: &label}},
			want:  `{"inner":{"flag":false,"label":"value"}}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()

			encoded, err := Marshal(test.input)
			require.NoError(t, err)
			require.JSONEq(t, test.want, string(encoded))

			var decoded marshalOuter

			require.NoError(t, Unmarshal(encoded, &decoded))
			require.Equal(t, test.input, decoded)
		})
	}
}

// the crash needed the shapes encoded in one process, so the cases run again in a fixed order rather
// than only as independent parallel subtests
func TestMarshalPointerToStructOfPointersInSequence(t *testing.T) {
	t.Parallel()

	flag := true

	for _, input := range []marshalOuter{
		{},
		{Inner: &marshalInner{}},
		{Inner: &marshalInner{Flag: &flag}},
	} {
		_, err := Marshal(input)
		require.NoError(t, err)
	}
}

// the decoder reports the members of an object as written: escaped names decoded, duplicates kept
func TestDecoderTokensKeepRawMemberNames(t *testing.T) {
	t.Parallel()

	decoder := NewDecoder(bytes.NewReader([]byte(`{"model":"a","mod\u0065l":"b","MODEL":{"x":[1]}}`)))

	token, err := decoder.Token()
	require.NoError(t, err)
	require.Equal(t, Delim('{'), token)

	var (
		names  []string
		values []string
	)

	for decoder.More() {
		token, err = decoder.Token()
		require.NoError(t, err)

		name, ok := token.(string)
		require.True(t, ok)

		var value RawMessage

		require.NoError(t, decoder.Decode(&value))

		names = append(names, name)
		values = append(values, string(value))
	}

	token, err = decoder.Token()
	require.NoError(t, err)
	require.Equal(t, Delim('}'), token)

	_, err = decoder.Token()
	require.ErrorIs(t, err, io.EOF)

	require.Equal(t, []string{"model", "model", "MODEL"}, names)
	require.Equal(t, []string{`"a"`, `"b"`, `{"x":[1]}`}, values)
}
