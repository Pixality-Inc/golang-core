package util

import (
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRoundFloat64ToPrecision(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		value     float64
		precision int
		want      float64
	}{
		{
			name:      "zero precision - round to integer",
			value:     3.14159,
			precision: 0,
			want:      3.0,
		},
		{
			name:      "one decimal place",
			value:     3.14159,
			precision: 1,
			want:      3.1,
		},
		{
			name:      "two decimal places",
			value:     3.14159,
			precision: 2,
			want:      3.14,
		},
		{
			name:      "three decimal places",
			value:     3.14159,
			precision: 3,
			want:      3.142,
		},
		{
			name:      "five decimal places",
			value:     3.14159265359,
			precision: 5,
			want:      3.14159,
		},
		{
			name:      "round up",
			value:     1.9999,
			precision: 2,
			want:      2.0,
		},
		{
			name:      "round down",
			value:     1.1111,
			precision: 2,
			want:      1.11,
		},
		{
			name:      "negative number",
			value:     -3.14159,
			precision: 2,
			want:      -3.14,
		},
		{
			name:      "zero value",
			value:     0.0,
			precision: 2,
			want:      0.0,
		},
		{
			name:      "already rounded",
			value:     5.5,
			precision: 1,
			want:      5.5,
		},
		{
			name:      "large number",
			value:     123456.789,
			precision: 2,
			want:      123456.79,
		},
		{
			name:      "small number",
			value:     0.000123456,
			precision: 6,
			want:      0.000123,
		},
		{
			name:      "exact half - rounds up",
			value:     2.5,
			precision: 0,
			want:      3.0,
		},
		{
			name:      "exact half - rounds up (another case)",
			value:     3.5,
			precision: 0,
			want:      4.0,
		},
		{
			name:      "very small precision value",
			value:     0.123456789,
			precision: 8,
			want:      0.12345679,
		},
		{
			name:      "negative precision (effectively rounds to left of decimal)",
			value:     12345.6789,
			precision: -1,
			want:      12350.0,
		},
		{
			name:      "negative precision with negative number",
			value:     -12345.6789,
			precision: -1,
			want:      -12350.0,
		},
	}

	for _, testcase := range tests {
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()

			got := RoundFloat64ToPrecision(testcase.value, testcase.precision)
			// Use InDelta for float comparison to handle floating point precision issues
			require.InDelta(t, testcase.want, got, 0.0000001)
		})
	}
}

func TestRoundFloat64ToInt64(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value float64
		want  int64
	}{
		{
			name:  "zero",
			value: 0,
			want:  0,
		},
		{
			name:  "positive number rounds down",
			value: 1.49,
			want:  1,
		},
		{
			name:  "positive half rounds away from zero",
			value: 1.5,
			want:  2,
		},
		{
			name:  "positive number rounds up",
			value: 1.51,
			want:  2,
		},
		{
			name:  "negative number rounds toward zero",
			value: -1.49,
			want:  -1,
		},
		{
			name:  "negative half rounds away from zero",
			value: -1.5,
			want:  -2,
		},
		{
			name:  "negative number rounds away from zero",
			value: -1.51,
			want:  -2,
		},
		{
			name:  "positive fraction below half rounds to zero",
			value: 0.49,
			want:  0,
		},
		{
			name:  "positive half rounds to one",
			value: 0.5,
			want:  1,
		},
		{
			name:  "negative fraction below half rounds to zero",
			value: -0.49,
			want:  0,
		},
		{
			name:  "negative half rounds to negative one",
			value: -0.5,
			want:  -1,
		},
		{
			name:  "positive integer remains unchanged",
			value: 42,
			want:  42,
		},
		{
			name:  "negative integer remains unchanged",
			value: -42,
			want:  -42,
		},
		{
			name:  "large exactly representable integer",
			value: 9_007_199_254_740_991,
			want:  9_007_199_254_740_991,
		},
	}

	for _, testcase := range tests {
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()

			got := RoundFloat64ToInt64(testcase.value)
			require.Equal(t, testcase.want, got)
		})
	}
}

func TestRoundFloat64ToPrecision_EdgeCases(t *testing.T) {
	t.Parallel()

	t.Run("infinity", func(t *testing.T) {
		t.Parallel()

		got := RoundFloat64ToPrecision(math.Inf(1), 2)
		require.True(t, math.IsInf(got, 1))
	})

	t.Run("negative infinity", func(t *testing.T) {
		t.Parallel()

		got := RoundFloat64ToPrecision(math.Inf(-1), 2)
		require.True(t, math.IsInf(got, -1))
	})

	t.Run("NaN", func(t *testing.T) {
		t.Parallel()

		got := RoundFloat64ToPrecision(math.NaN(), 2)
		require.True(t, math.IsNaN(got))
	})
}
