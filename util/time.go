package util

import (
	"fmt"
	"strings"
	"time"
)

func MaxDuration(durations ...time.Duration) time.Duration {
	return MaxDurationSlice(durations)
}

func MaxDurationSlice(durations []time.Duration) time.Duration {
	if len(durations) == 0 {
		return 0
	}

	maxDuration := durations[0]

	for _, d := range durations[1:] {
		if d > maxDuration {
			maxDuration = d
		}
	}

	return maxDuration
}

func FormatDuration(duration time.Duration) string {
	days := int64(duration / (24 * time.Hour))
	duration %= 24 * time.Hour

	hours := int64(duration / time.Hour)
	duration %= time.Hour

	minutes := int64(duration / time.Minute)
	duration %= time.Minute

	seconds := int64(duration / time.Second)
	duration %= time.Second

	millis := int64(duration / time.Millisecond)

	var parts []string

	if days > 0 {
		parts = append(parts, fmt.Sprintf("%dd", days))
	}

	if hours > 0 {
		parts = append(parts, fmt.Sprintf("%dh", hours))
	}

	if minutes > 0 {
		parts = append(parts, fmt.Sprintf("%dm", minutes))
	}

	if seconds > 0 {
		parts = append(parts, fmt.Sprintf("%ds", seconds))
	}

	if millis > 0 {
		parts = append(parts, fmt.Sprintf("%dms", millis))
	}

	if len(parts) == 0 {
		return "0ms"
	}

	return strings.Join(parts, " ")
}
