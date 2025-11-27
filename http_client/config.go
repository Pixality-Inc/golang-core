package http_client

import (
	"crypto/tls"
	"time"

	"github.com/pixality-inc/golang-core/circuit_breaker"
	"github.com/pixality-inc/golang-core/retry"
)

const (
	DefaultMaxConnsPerHost     = 512
	DefaultMaxIdleConnDuration = 90 * time.Second
	DefaultReadBufferSize      = 4096
	DefaultWriteBufferSize     = 4096
	DefaultMaxResponseBodySize = 10 * 1024 * 1024 // 10 MB
	DefaultMaxConnDuration     = 0                // no limit
)

const (
	TLSVersion10 uint16 = tls.VersionTLS10
	TLSVersion11 uint16 = tls.VersionTLS11
	TLSVersion12 uint16 = tls.VersionTLS12
	TLSVersion13 uint16 = tls.VersionTLS13
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
	RetryPolicy() retry.Policy
	ReadBufferSize() int
	WriteBufferSize() int
	MaxResponseBodySize() int
	MaxConnDuration() time.Duration
	StreamResponseBody() bool
	TLSMinVersion() uint16
	TLSMaxVersion() uint16
	TLSServerName() string
	TLSRootCAFile() string
	TLSClientCertFile() string
	TLSClientKeyFile() string
	CircuitBreaker() circuit_breaker.Config
}

type ConfigYaml struct {
	BaseUrlValue             string                      `env:"BASE_URL"               yaml:"base_url"`
	InsecureSkipVerifyValue  bool                        `env:"INSECURE_SKIP_VERIFY"   yaml:"insecure_skip_verify"`
	TimeoutValue             time.Duration               `env:"TIMEOUT"                yaml:"timeout"`
	BaseHeadersValue         Headers                     `env:"BASE_HEADERS"           yaml:"base_headers"`
	UseRequestIdValue        bool                        `env:"USE_REQUEST_ID"         yaml:"use_request_id"`
	NameValue                string                      `env:"NAME"                   yaml:"name"`
	MaxConnsPerHostValue     int                         `env:"MAX_CONNS_PER_HOST"     yaml:"max_conns_per_host"`
	MaxIdleConnDurationValue time.Duration               `env:"MAX_IDLE_CONN_DURATION" yaml:"max_idle_conn_duration"`
	ReadTimeoutValue         time.Duration               `env:"READ_TIMEOUT"           yaml:"read_timeout"`
	WriteTimeoutValue        time.Duration               `env:"WRITE_TIMEOUT"          yaml:"write_timeout"`
	MaxConnWaitTimeoutValue  time.Duration               `env:"MAX_CONN_WAIT_TIMEOUT"  yaml:"max_conn_wait_timeout"`
	RetryPolicyValue         *retry.ConfigYaml           `env-prefix:"RETRY_POLICY"    yaml:"retry_policy"`
	ReadBufferSizeValue      int                         `env:"READ_BUFFER_SIZE"       yaml:"read_buffer_size"`
	WriteBufferSizeValue     int                         `env:"WRITE_BUFFER_SIZE"      yaml:"write_buffer_size"`
	MaxResponseBodySizeValue int                         `env:"MAX_RESPONSE_BODY_SIZE" yaml:"max_response_body_size"`
	MaxConnDurationValue     time.Duration               `env:"MAX_CONN_DURATION"      yaml:"max_conn_duration"`
	StreamResponseBodyValue  bool                        `env:"STREAM_RESPONSE_BODY"   yaml:"stream_response_body"`
	TLSMinVersionValue       uint16                      `env:"TLS_MIN_VERSION"        yaml:"tls_min_version"`
	TLSMaxVersionValue       uint16                      `env:"TLS_MAX_VERSION"        yaml:"tls_max_version"`
	TLSServerNameValue       string                      `env:"TLS_SERVER_NAME"        yaml:"tls_server_name"`
	TLSRootCAFileValue       string                      `env:"TLS_ROOT_CA_FILE"       yaml:"tls_root_ca_file"`
	TLSClientCertFileValue   string                      `env:"TLS_CLIENT_CERT_FILE"   yaml:"tls_client_cert_file"`
	TLSClientKeyFileValue    string                      `env:"TLS_CLIENT_KEY_FILE"    yaml:"tls_client_key_file"`
	CircuitBreakerValue      *circuit_breaker.ConfigYaml `env-prefix:"CIRCUIT_BREAKER" yaml:"circuit_breaker"`
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

func (c *ConfigYaml) RetryPolicy() retry.Policy {
	if c.RetryPolicyValue == nil {
		return nil
	}

	return c.RetryPolicyValue
}

func (c *ConfigYaml) ReadBufferSize() int {
	if c.ReadBufferSizeValue == 0 {
		return DefaultReadBufferSize
	}

	return c.ReadBufferSizeValue
}

func (c *ConfigYaml) WriteBufferSize() int {
	if c.WriteBufferSizeValue == 0 {
		return DefaultWriteBufferSize
	}

	return c.WriteBufferSizeValue
}

func (c *ConfigYaml) MaxResponseBodySize() int {
	if c.MaxResponseBodySizeValue == 0 {
		return DefaultMaxResponseBodySize
	}

	return c.MaxResponseBodySizeValue
}

func (c *ConfigYaml) MaxConnDuration() time.Duration {
	if c.MaxConnDurationValue == 0 {
		return DefaultMaxConnDuration
	}

	return c.MaxConnDurationValue
}

func (c *ConfigYaml) StreamResponseBody() bool {
	return c.StreamResponseBodyValue
}

func (c *ConfigYaml) TLSMinVersion() uint16 {
	return c.TLSMinVersionValue
}

func (c *ConfigYaml) TLSMaxVersion() uint16 {
	return c.TLSMaxVersionValue
}

func (c *ConfigYaml) TLSServerName() string {
	return c.TLSServerNameValue
}

func (c *ConfigYaml) TLSRootCAFile() string {
	return c.TLSRootCAFileValue
}

func (c *ConfigYaml) TLSClientCertFile() string {
	return c.TLSClientCertFileValue
}

func (c *ConfigYaml) TLSClientKeyFile() string {
	return c.TLSClientKeyFileValue
}

func (c *ConfigYaml) CircuitBreaker() circuit_breaker.Config {
	if c.CircuitBreakerValue == nil {
		return nil
	}

	return c.CircuitBreakerValue
}
