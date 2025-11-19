package metrics

import "time"

type SummaryOptions interface {
	Objectives() map[float64]float64
	MaxAge() time.Duration
	AgeBuckets() uint32
	BufCap() uint32
}

type SummaryOptionsImpl struct {
	objectives map[float64]float64
	maxAge     time.Duration
	ageBuckets uint32
	bufCap     uint32
}

func NewSummaryOptions() *SummaryOptionsImpl {
	return &SummaryOptionsImpl{
		objectives: nil,
		maxAge:     0,
		ageBuckets: 0,
		bufCap:     0,
	}
}

func (s *SummaryOptionsImpl) WithObjectives(objectives map[float64]float64) *SummaryOptionsImpl {
	s.objectives = objectives

	return s
}

func (s *SummaryOptionsImpl) WithMaxAge(maxAge time.Duration) *SummaryOptionsImpl {
	s.maxAge = maxAge

	return s
}

func (s *SummaryOptionsImpl) WithAgeBuckets(buckets uint32) *SummaryOptionsImpl {
	s.ageBuckets = buckets

	return s
}

func (s *SummaryOptionsImpl) WithBufCap(bufCap uint32) *SummaryOptionsImpl {
	s.bufCap = bufCap

	return s
}

func (s *SummaryOptionsImpl) Objectives() map[float64]float64 {
	return s.objectives
}

func (s *SummaryOptionsImpl) MaxAge() time.Duration {
	return s.maxAge
}

func (s *SummaryOptionsImpl) AgeBuckets() uint32 {
	return s.ageBuckets
}

func (s *SummaryOptionsImpl) BufCap() uint32 {
	return s.bufCap
}
