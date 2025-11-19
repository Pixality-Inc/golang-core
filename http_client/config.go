package http_client

import "time"

type Config interface {
	BaseUrl() string
	InsecureSkipVerify() bool
	Timeout() time.Duration
	BaseHeaders() Headers
	UseRequestId() bool
}

type ConfigYaml struct {
	BaseUrlValue            string        `env:"BASE_URL"             yaml:"base_url"`
	InsecureSkipVerifyValue bool          `env:"INSECURE_SKIP_VERIFY" yaml:"insecure_skip_verify"`
	TimeoutValue            time.Duration `env:"TIMEOUT"              yaml:"timeout"`
	BaseHeadersValue        Headers       `env:"BASE_HEADERS"         yaml:"base_headers"`
	UseRequestIdValue       bool          `env:"USE_REQUEST_ID"       yaml:"use_request_id"`
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
