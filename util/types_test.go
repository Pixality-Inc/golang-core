package util

import (
	"testing"

	"github.com/google/uuid"
	"github.com/pixality-inc/golang-core/json"
	"github.com/stretchr/testify/require"
)

type testIdType uuid.UUID

var emptyTestIdType = testIdType(uuid.Nil)

func (v *testIdType) UnmarshalJSON(data []byte) error {
	value, err := UnmarshalJsonToId(data, emptyTestIdType, func(uuidValue uuid.UUID) testIdType {
		return testIdType(uuidValue)
	})
	if err != nil {
		return err
	}

	*v = value

	return nil
}

func TestUnmarshalJsonToId(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		data    []byte
		want    testIdType
		wantErr error
	}{
		{
			name:    "byte_array",
			data:    []byte(`[32, 99, 203, 69, 164, 115, 76, 87, 157, 103, 175, 76, 250, 191, 15, 187]`),
			want:    testIdType(uuid.UUID{32, 99, 203, 69, 164, 115, 76, 87, 157, 103, 175, 76, 250, 191, 15, 187}),
			wantErr: nil,
		},
		{
			name:    "bad_byte_array",
			data:    []byte(`[69, 69, 69]`),
			want:    emptyTestIdType,
			wantErr: ErrUnmarshalByteArray,
		},
		{
			name:    "string",
			data:    []byte(`"dc580a7d-7ab6-4f27-9f1e-06941c7c13c6"`),
			want:    testIdType(uuid.MustParse("dc580a7d-7ab6-4f27-9f1e-06941c7c13c6")),
			wantErr: nil,
		},
		{
			name:    "bad_string",
			data:    []byte(`"test"`),
			want:    emptyTestIdType,
			wantErr: ErrUnmarshalString,
		},
		{
			name:    "bool",
			data:    []byte("true"),
			want:    emptyTestIdType,
			wantErr: ErrUnmarshal,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			var testId testIdType

			err := json.Unmarshal(testCase.data, &testId)
			if testCase.wantErr != nil {
				require.ErrorIs(t, err, testCase.wantErr)
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, testCase.want, testId)
		})
	}
}
