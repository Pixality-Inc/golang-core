package retry

import "time"

type Policy interface {
	MaxAttempts() int
	InitialInterval() time.Duration
	BackoffCoefficient() float64
	MaxInterval() time.Duration
}

type PolicyImpl struct {
	MaxAttemptsValue        int           `yaml:"max_attempts"`
	InitialIntervalValue    time.Duration `yaml:"initial_interval"`
	BackoffCoefficientValue float64       `yaml:"backoff_coefficient"`
	MaxIntervalValue        time.Duration `yaml:"max_interval"`
}

func (p *PolicyImpl) MaxAttempts() int {
	return p.MaxAttemptsValue
}

func (p *PolicyImpl) InitialInterval() time.Duration {
	return p.InitialIntervalValue
}

func (p *PolicyImpl) BackoffCoefficient() float64 {
	return p.BackoffCoefficientValue
}

func (p *PolicyImpl) MaxInterval() time.Duration {
	return p.MaxIntervalValue
}

type Config struct {
	MaxAttemptsValue        int           `yaml:"max_attempts"`
	InitialIntervalValue    time.Duration `yaml:"initial_interval"`
	BackoffCoefficientValue float64       `yaml:"backoff_coefficient"`
	MaxIntervalValue        time.Duration `yaml:"max_interval"`
}

func (c *Config) MaxAttempts() int {
	return c.MaxAttemptsValue
}

func (c *Config) InitialInterval() time.Duration {
	return c.InitialIntervalValue
}

func (c *Config) BackoffCoefficient() float64 {
	return c.BackoffCoefficientValue
}

func (c *Config) MaxInterval() time.Duration {
	return c.MaxIntervalValue
}

type PolicyOption func(*PolicyImpl)

func WithMaxAttempts(maxAttempts int) PolicyOption {
	return func(p *PolicyImpl) {
		p.MaxAttemptsValue = maxAttempts
	}
}

func WithInitialInterval(interval time.Duration) PolicyOption {
	return func(p *PolicyImpl) {
		p.InitialIntervalValue = interval
	}
}

func WithBackoffCoefficient(coefficient float64) PolicyOption {
	return func(p *PolicyImpl) {
		p.BackoffCoefficientValue = coefficient
	}
}

func WithMaxInterval(interval time.Duration) PolicyOption {
	return func(p *PolicyImpl) {
		p.MaxIntervalValue = interval
	}
}

func NewPolicy(opts ...PolicyOption) Policy {
	policy := &PolicyImpl{
		MaxAttemptsValue:        3,
		InitialIntervalValue:    100 * time.Millisecond,
		BackoffCoefficientValue: 2.0,
		MaxIntervalValue:        5 * time.Second,
	}

	for _, opt := range opts {
		opt(policy)
	}

	return policy
}
