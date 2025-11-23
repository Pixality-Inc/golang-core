package http_client

import "time"

const (
	DefaultMaxConnsPerHost     = 512
	DefaultMaxIdleConnDuration = 90 * time.Second
	DefaultReadTimeout         = 0
	DefaultWriteTimeout        = 0
)

type Config interface {
	BaseUrl() string
	InsecureSkipVerify() bool
	Timeout() time.Duration
	BaseHeaders() Headers
	UseRequestId() bool
	Name() string
	MaxConnsPerHost() int
	MaxIdleConnDuration() time.Duration
	ReadTimeout() time.Duration
	WriteTimeout() time.Duration
	MaxConnWaitTimeout() time.Duration
	RetryPolicy() *RetryPolicy
}

type RetryPolicy struct {
	MaxAttempts        int           `yaml:"max_attempts"`
	InitialInterval    time.Duration `yaml:"initial_interval"`
	BackoffCoefficient float64       `yaml:"backoff_coefficient"`
	MaxInterval        time.Duration `yaml:"max_interval"`
}

type ConfigYaml struct {
	BaseUrlValue             string        `env:"BASE_URL"             yaml:"base_url"`
	InsecureSkipVerifyValue  bool          `env:"INSECURE_SKIP_VERIFY" yaml:"insecure_skip_verify"`
	TimeoutValue             time.Duration `env:"TIMEOUT"              yaml:"timeout"`
	BaseHeadersValue         Headers       `env:"BASE_HEADERS"         yaml:"base_headers"`
	UseRequestIdValue        bool          `env:"USE_REQUEST_ID"       yaml:"use_request_id"`
	NameValue                string        `env:"NAME"                 yaml:"name"`
	MaxConnsPerHostValue     int           `env:"MAX_CONNS_PER_HOST"   yaml:"max_conns_per_host"`
	MaxIdleConnDurationValue time.Duration `env:"MAX_IDLE_CONN_DURATION"    yaml:"max_idle_conn_duration"`
	ReadTimeoutValue         time.Duration `env:"READ_TIMEOUT"              yaml:"read_timeout"`
	WriteTimeoutValue        time.Duration `env:"WRITE_TIMEOUT"             yaml:"write_timeout"`
	MaxConnWaitTimeoutValue  time.Duration `env:"MAX_CONN_WAIT_TIMEOUT"     yaml:"max_conn_wait_timeout"`
	RetryPolicyValue         *RetryPolicy  `env:"RETRY_POLICY"              yaml:"retry_policy"`
}

func (c *ConfigYaml) BaseUrl() string {
	return c.BaseUrlValue
}

func (c *ConfigYaml) InsecureSkipVerify() bool {
	return c.InsecureSkipVerifyValue
}

func (c *ConfigYaml) Timeout() time.Duration {
	return c.TimeoutValue
}

func (c *ConfigYaml) BaseHeaders() Headers {
	return c.BaseHeadersValue
}

func (c *ConfigYaml) UseRequestId() bool {
	return c.UseRequestIdValue
}

func (c *ConfigYaml) Name() string {
	if c.NameValue == "" {
		return "http_client"
	}
	return c.NameValue
}

func (c *ConfigYaml) MaxConnsPerHost() int {
	if c.MaxConnsPerHostValue == 0 {
		return DefaultMaxConnsPerHost
	}
	return c.MaxConnsPerHostValue
}

func (c *ConfigYaml) MaxIdleConnDuration() time.Duration {
	if c.MaxIdleConnDurationValue == 0 {
		return DefaultMaxIdleConnDuration
	}
	return c.MaxIdleConnDurationValue
}

func (c *ConfigYaml) ReadTimeout() time.Duration {
	if c.ReadTimeoutValue == 0 {
		return c.TimeoutValue
	}
	return c.ReadTimeoutValue
}

func (c *ConfigYaml) WriteTimeout() time.Duration {
	if c.WriteTimeoutValue == 0 {
		return c.TimeoutValue
	}
	return c.WriteTimeoutValue
}

func (c *ConfigYaml) MaxConnWaitTimeout() time.Duration {
	return c.MaxConnWaitTimeoutValue
}

func (c *ConfigYaml) RetryPolicy() *RetryPolicy {
	return c.RetryPolicyValue
}
