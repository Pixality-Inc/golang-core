package json

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAsJsonArray(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		input []int
	}{
		{
			name:  "empty slice",
			input: []int{},
		},
		{
			name:  "single element",
			input: []int{1},
		},
		{
			name:  "multiple elements",
			input: []int{1, 2, 3, 4, 5},
		},
	}

	for _, testcase := range tests {
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()

			got := AsJsonArray(testcase.input)
			require.NotNil(t, got)
			require.Len(t, got, len(testcase.input))
		})
	}
}
