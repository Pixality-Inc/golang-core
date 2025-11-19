package util

import "math"

func RoundFloat64ToPrecision(f float64, precision int) float64 {
	multiplier := math.Pow(10, float64(precision))

	return math.Round(f*multiplier) / multiplier
}
