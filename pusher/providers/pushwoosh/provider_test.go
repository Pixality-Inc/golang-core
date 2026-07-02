package pushwoosh

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_getTimezoneOffsetSeconds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		timezone string
		expected int
	}{
		{
			name:     "UTC",
			timezone: "UTC",
			expected: 0,
		},
		{
			name:     "positive offset",
			timezone: "Etc/GMT-5",
			expected: 18000,
		},
		{
			name:     "negative offset",
			timezone: "Etc/GMT+3",
			expected: -10800,
		},
		{
			name:     "moscow",
			timezone: "europe/moscow",
			expected: 10800,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			offset, err := getTimezoneOffsetSeconds(testCase.timezone)
			require.NoError(t, err)
			require.Equal(t, testCase.expected, offset)
		})
	}
}

func Test_getTimezoneOffsetSeconds_invalidTimezone(t *testing.T) {
	t.Parallel()

	offset, err := getTimezoneOffsetSeconds("Unknown/Timezone")
	require.Error(t, err)
	require.Zero(t, offset)
}
