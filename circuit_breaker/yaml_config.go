package circuit_breaker

import "time"

type ConfigYaml struct {
	EnabledValue             bool          `env:"ENABLED"              yaml:"enabled"`
	NameValue                string        `env:"NAME"                 yaml:"name"`
	MaxRequestsValue         uint32        `env:"MAX_REQUESTS"         yaml:"max_requests"`
	IntervalValue            time.Duration `env:"INTERVAL"             yaml:"interval"`
	TimeoutValue             time.Duration `env:"TIMEOUT"              yaml:"timeout"`
	ConsecutiveFailuresValue uint32        `env:"CONSECUTIVE_FAILURES" yaml:"consecutive_failures"`
	BucketPeriodValue        time.Duration `env:"BUCKET_PERIOD"        yaml:"bucket_period"`
}

func (c *ConfigYaml) Enabled() bool {
	return c.EnabledValue
}

func (c *ConfigYaml) Name() string {
	return c.NameValue
}

func (c *ConfigYaml) MaxRequests() uint32 {
	return c.MaxRequestsValue
}

func (c *ConfigYaml) Interval() time.Duration {
	return c.IntervalValue
}

func (c *ConfigYaml) Timeout() time.Duration {
	return c.TimeoutValue
}

func (c *ConfigYaml) ConsecutiveFailures() uint32 {
	return c.ConsecutiveFailuresValue
}

func (c *ConfigYaml) BucketPeriod() time.Duration {
	return c.BucketPeriodValue
}
