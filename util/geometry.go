package util

import "github.com/paulmach/orb"

func NewPoint(lon float64, lat float64) orb.Point {
	return orb.Point{lon, lat}
}
