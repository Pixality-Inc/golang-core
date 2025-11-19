package util

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMaxDuration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		durations []time.Duration
		want      time.Duration
	}{
		{
			name:      "empty",
			durations: []time.Duration{},
			want:      0,
		},
		{
			name:      "single duration",
			durations: []time.Duration{5 * time.Second},
			want:      5 * time.Second,
		},
		{
			name:      "multiple durations",
			durations: []time.Duration{3 * time.Second, 10 * time.Second, 5 * time.Second},
			want:      10 * time.Second,
		},
		{
			name:      "max is first",
			durations: []time.Duration{10 * time.Second, 3 * time.Second, 5 * time.Second},
			want:      10 * time.Second,
		},
		{
			name:      "max is last",
			durations: []time.Duration{3 * time.Second, 5 * time.Second, 10 * time.Second},
			want:      10 * time.Second,
		},
		{
			name:      "negative durations",
			durations: []time.Duration{-5 * time.Second, -10 * time.Second, -3 * time.Second},
			want:      -3 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := MaxDuration(tt.durations...)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestMaxDurationSlice(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		durations []time.Duration
		want      time.Duration
	}{
		{
			name:      "empty slice",
			durations: []time.Duration{},
			want:      0,
		},
		{
			name:      "single element",
			durations: []time.Duration{5 * time.Minute},
			want:      5 * time.Minute,
		},
		{
			name:      "multiple elements",
			durations: []time.Duration{2 * time.Hour, 30 * time.Minute, 90 * time.Minute},
			want:      2 * time.Hour,
		},
		{
			name:      "all equal",
			durations: []time.Duration{1 * time.Second, 1 * time.Second, 1 * time.Second},
			want:      1 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := MaxDurationSlice(tt.durations)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestFormatDuration(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{
			name:     "zero duration",
			duration: 0,
			want:     "",
		},
		{
			name:     "only milliseconds",
			duration: 500 * time.Millisecond,
			want:     "500ms",
		},
		{
			name:     "only seconds",
			duration: 30 * time.Second,
			want:     "30s",
		},
		{
			name:     "only minutes",
			duration: 15 * time.Minute,
			want:     "15m",
		},
		{
			name:     "only hours",
			duration: 2 * time.Hour,
			want:     "2h",
		},
		{
			name:     "only days",
			duration: 3 * 24 * time.Hour,
			want:     "3d",
		},
		{
			name:     "seconds and milliseconds",
			duration: 5*time.Second + 250*time.Millisecond,
			want:     "5s 250ms",
		},
		{
			name:     "minutes and seconds",
			duration: 2*time.Minute + 30*time.Second,
			want:     "2m 30s",
		},
		{
			name:     "hours and minutes",
			duration: 1*time.Hour + 45*time.Minute,
			want:     "1h 45m",
		},
		{
			name:     "days and hours",
			duration: 2*24*time.Hour + 6*time.Hour,
			want:     "2d 6h",
		},
		{
			name:     "complex duration",
			duration: 1*24*time.Hour + 2*time.Hour + 30*time.Minute + 45*time.Second + 123*time.Millisecond,
			want:     "1d 2h 30m 45s 123ms",
		},
		{
			name:     "all components except days",
			duration: 3*time.Hour + 15*time.Minute + 20*time.Second + 500*time.Millisecond,
			want:     "3h 15m 20s 500ms",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := FormatDuration(tt.duration)
			require.Equal(t, tt.want, got)
		})
	}
}
