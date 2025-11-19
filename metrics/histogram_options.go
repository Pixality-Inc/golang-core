package metrics

import "time"

type HistogramOptions interface {
	Buckets() []float64
	NativeHistogramBucketFactor() float64
	NativeHistogramZeroThreshold() float64
	NativeHistogramMaxBucketNumber() uint32
	NativeHistogramMinResetDuration() time.Duration
	NativeHistogramMaxZeroThreshold() float64
	NativeHistogramMaxExemplars() int
	NativeHistogramExemplarTTL() time.Duration
}

type HistogramOptionsImpl struct {
	buckets                         []float64
	nativeHistogramBucketFactor     float64
	nativeHistogramZeroThreshold    float64
	nativeHistogramMaxBucketNumber  uint32
	nativeHistogramMinResetDuration time.Duration
	nativeHistogramMaxZeroThreshold float64
	nativeHistogramMaxExemplars     int
	nativeHistogramExemplarTTL      time.Duration
}

func NewHistogramOptions() *HistogramOptionsImpl {
	return &HistogramOptionsImpl{
		buckets:                         nil,
		nativeHistogramBucketFactor:     0,
		nativeHistogramZeroThreshold:    0,
		nativeHistogramMaxBucketNumber:  0,
		nativeHistogramMinResetDuration: 0,
		nativeHistogramMaxZeroThreshold: 0,
		nativeHistogramMaxExemplars:     0,
		nativeHistogramExemplarTTL:      0,
	}
}

func (o *HistogramOptionsImpl) WithBuckets(buckets []float64) *HistogramOptionsImpl {
	o.buckets = buckets

	return o
}

func (o *HistogramOptionsImpl) WithNativeHistogramBucketFactor(factor float64) *HistogramOptionsImpl {
	o.nativeHistogramBucketFactor = factor

	return o
}

func (o *HistogramOptionsImpl) WithNativeHistogramZeroThreshold(zeroThreshold float64) *HistogramOptionsImpl {
	o.nativeHistogramZeroThreshold = zeroThreshold

	return o
}

func (o *HistogramOptionsImpl) WithNativeHistogramMaxBucketNumber(maxValue uint32) *HistogramOptionsImpl {
	o.nativeHistogramMaxBucketNumber = maxValue

	return o
}

func (o *HistogramOptionsImpl) WithNativeHistogramMinResetDuration(minResetDuration time.Duration) *HistogramOptionsImpl {
	o.nativeHistogramMinResetDuration = minResetDuration

	return o
}

func (o *HistogramOptionsImpl) WithNativeHistogramMaxZeroThreshold(maxValue float64) *HistogramOptionsImpl {
	o.nativeHistogramMaxZeroThreshold = maxValue

	return o
}

func (o *HistogramOptionsImpl) WithNativeHistogramMaxExemplars(maxValue int) *HistogramOptionsImpl {
	o.nativeHistogramMaxExemplars = maxValue

	return o
}

func (o *HistogramOptionsImpl) WithNativeHistogramExemplarTTL(ttl time.Duration) *HistogramOptionsImpl {
	o.nativeHistogramExemplarTTL = ttl

	return o
}

func (o *HistogramOptionsImpl) Buckets() []float64 {
	return o.buckets
}

func (o *HistogramOptionsImpl) NativeHistogramBucketFactor() float64 {
	return o.nativeHistogramBucketFactor
}

func (o *HistogramOptionsImpl) NativeHistogramZeroThreshold() float64 {
	return o.nativeHistogramZeroThreshold
}

func (o *HistogramOptionsImpl) NativeHistogramMaxBucketNumber() uint32 {
	return o.nativeHistogramMaxBucketNumber
}

func (o *HistogramOptionsImpl) NativeHistogramMinResetDuration() time.Duration {
	return o.nativeHistogramMinResetDuration
}

func (o *HistogramOptionsImpl) NativeHistogramMaxZeroThreshold() float64 {
	return o.nativeHistogramMaxZeroThreshold
}

func (o *HistogramOptionsImpl) NativeHistogramMaxExemplars() int {
	return o.nativeHistogramMaxExemplars
}

func (o *HistogramOptionsImpl) NativeHistogramExemplarTTL() time.Duration {
	return o.nativeHistogramExemplarTTL
}
