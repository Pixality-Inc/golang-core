package retry

import "time"

type Policy interface {
	Enabled() bool
	MaxAttempts() int
	InitialInterval() time.Duration
	BackoffCoefficient() float64
	MaxInterval() time.Duration
	RetryNonIdempotent() bool
}

type PolicyImpl struct {
	EnabledValue            bool          `env:"ENABLED"              yaml:"enabled"`
	MaxAttemptsValue        int           `env:"MAX_ATTEMPTS"         yaml:"max_attempts"`
	InitialIntervalValue    time.Duration `env:"INITIAL_INTERVAL"     yaml:"initial_interval"`
	BackoffCoefficientValue float64       `env:"BACKOFF_COEFFICIENT"  yaml:"backoff_coefficient"`
	MaxIntervalValue        time.Duration `env:"MAX_INTERVAL"         yaml:"max_interval"`
	RetryNonIdempotentValue bool          `env:"RETRY_NON_IDEMPOTENT" yaml:"retry_non_idempotent"`
}

func (p *PolicyImpl) Enabled() bool {
	return p.EnabledValue
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

func (p *PolicyImpl) RetryNonIdempotent() bool {
	return p.RetryNonIdempotentValue
}

type ConfigYaml struct {
	EnabledValue            bool          `env:"ENABLED"              yaml:"enabled"`
	MaxAttemptsValue        int           `env:"MAX_ATTEMPTS"         yaml:"max_attempts"`
	InitialIntervalValue    time.Duration `env:"INITIAL_INTERVAL"     yaml:"initial_interval"`
	BackoffCoefficientValue float64       `env:"BACKOFF_COEFFICIENT"  yaml:"backoff_coefficient"`
	MaxIntervalValue        time.Duration `env:"MAX_INTERVAL"         yaml:"max_interval"`
	RetryNonIdempotentValue bool          `env:"RETRY_NON_IDEMPOTENT" yaml:"retry_non_idempotent"`
}

func (c *ConfigYaml) Enabled() bool {
	return c.EnabledValue
}

func (c *ConfigYaml) MaxAttempts() int {
	return c.MaxAttemptsValue
}

func (c *ConfigYaml) InitialInterval() time.Duration {
	return c.InitialIntervalValue
}

func (c *ConfigYaml) BackoffCoefficient() float64 {
	return c.BackoffCoefficientValue
}

func (c *ConfigYaml) MaxInterval() time.Duration {
	return c.MaxIntervalValue
}

func (c *ConfigYaml) RetryNonIdempotent() bool {
	return c.RetryNonIdempotentValue
}

type PolicyOption func(*PolicyImpl)

func WithEnabled(enabled bool) PolicyOption {
	return func(p *PolicyImpl) {
		p.EnabledValue = enabled
	}
}

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

func WithRetryNonIdempotent(retry bool) PolicyOption {
	return func(p *PolicyImpl) {
		p.RetryNonIdempotentValue = retry
	}
}

func NewPolicy(opts ...PolicyOption) Policy {
	policy := &PolicyImpl{
		EnabledValue:            false,
		MaxAttemptsValue:        3,
		InitialIntervalValue:    100 * time.Millisecond,
		BackoffCoefficientValue: 2.0,
		MaxIntervalValue:        5 * time.Second,
		RetryNonIdempotentValue: false,
	}

	for _, opt := range opts {
		opt(policy)
	}

	return policy
}
