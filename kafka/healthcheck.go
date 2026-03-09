package kafka

import (
	"context"
	"time"
)

const defaultHealthcheckTimeout = 2 * time.Second

type Pingable interface {
	IsConnected() bool
	Ping(ctx context.Context) error
}

type HealthcheckOption func(*HealthcheckService)

func WithHealthcheckTimeout(timeout time.Duration) HealthcheckOption {
	return func(svc *HealthcheckService) {
		svc.timeout = timeout
	}
}

type HealthcheckService struct {
	target  Pingable
	timeout time.Duration
}

func NewHealthcheckService(target Pingable, opts ...HealthcheckOption) *HealthcheckService {
	svc := &HealthcheckService{
		target:  target,
		timeout: defaultHealthcheckTimeout,
	}

	for _, opt := range opts {
		opt(svc)
	}

	return svc
}

func (h *HealthcheckService) IsOK() bool {
	if !h.target.IsConnected() {
		return true
	}

	ctx, cancel := context.WithTimeout(context.Background(), h.timeout)
	defer cancel()

	return h.target.Ping(ctx) == nil
}
