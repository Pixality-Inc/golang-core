package util

import (
	"testing"

	"github.com/paulmach/orb"
	"github.com/stretchr/testify/require"
)

func TestNewPoint(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		lon  float64
		lat  float64
		want orb.Point
	}{
		{
			name: "zero coordinates",
			lon:  0.0,
			lat:  0.0,
			want: orb.Point{0.0, 0.0},
		},
		{
			name: "positive coordinates",
			lon:  10.5,
			lat:  20.3,
			want: orb.Point{10.5, 20.3},
		},
		{
			name: "negative coordinates",
			lon:  -10.5,
			lat:  -20.3,
			want: orb.Point{-10.5, -20.3},
		},
		{
			name: "mixed coordinates",
			lon:  -122.4194,
			lat:  37.7749,
			want: orb.Point{-122.4194, 37.7749},
		},
		{
			name: "extreme longitude",
			lon:  180.0,
			lat:  0.0,
			want: orb.Point{180.0, 0.0},
		},
		{
			name: "extreme latitude",
			lon:  0.0,
			lat:  90.0,
			want: orb.Point{0.0, 90.0},
		},
		{
			name: "negative extreme longitude",
			lon:  -180.0,
			lat:  0.0,
			want: orb.Point{-180.0, 0.0},
		},
		{
			name: "negative extreme latitude",
			lon:  0.0,
			lat:  -90.0,
			want: orb.Point{0.0, -90.0},
		},
		{
			name: "New York coordinates",
			lon:  -74.0060,
			lat:  40.7128,
			want: orb.Point{-74.0060, 40.7128},
		},
		{
			name: "London coordinates",
			lon:  -0.1278,
			lat:  51.5074,
			want: orb.Point{-0.1278, 51.5074},
		},
		{
			name: "Tokyo coordinates",
			lon:  139.6917,
			lat:  35.6895,
			want: orb.Point{139.6917, 35.6895},
		},
		{
			name: "high precision coordinates",
			lon:  12.3456789,
			lat:  98.7654321,
			want: orb.Point{12.3456789, 98.7654321},
		},
	}

	for _, testcase := range tests {
		t.Run(testcase.name, func(t *testing.T) {
			t.Parallel()

			got := NewPoint(testcase.lon, testcase.lat)
			require.Equal(t, testcase.want, got)
			require.InDelta(t, testcase.lon, got.Lon(), 1e-6)
			require.InDelta(t, testcase.lat, got.Lat(), 1e-6)
		})
	}
}
