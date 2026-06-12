package http_client

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pixality-inc/golang-core/circuit_breaker"
	"github.com/pixality-inc/golang-core/retry"
)

func TestConfigYaml_Defaults(t *testing.T) {
	t.Parallel()

	cfg := &ConfigYaml{}

	assert.Empty(t, cfg.BaseUrl())
	assert.False(t, cfg.InsecureSkipVerify())
	assert.Equal(t, time.Duration(0), cfg.Timeout())
	assert.Nil(t, cfg.BaseHeaders())
	assert.False(t, cfg.UseRequestId())
	assert.Equal(t, "http_client", cfg.Name())
	assert.Equal(t, DefaultMaxConnsPerHost, cfg.MaxConnsPerHost())
	assert.Equal(t, DefaultMaxIdleConnDuration, cfg.MaxIdleConnDuration())
	assert.Equal(t, time.Duration(0), cfg.ReadTimeout())
	assert.Equal(t, time.Duration(0), cfg.WriteTimeout())
	assert.Equal(t, time.Duration(0), cfg.MaxConnWaitTimeout())
	assert.Equal(t, DefaultReadBufferSize, cfg.ReadBufferSize())
	assert.Equal(t, DefaultWriteBufferSize, cfg.WriteBufferSize())
	assert.Equal(t, 0, cfg.MaxResponseBodySize())
	assert.Equal(t, time.Duration(DefaultMaxConnDuration), cfg.MaxConnDuration())
	assert.False(t, cfg.StreamResponseBody())
	assert.False(t, cfg.FollowRedirects())
	assert.Equal(t, uint16(0), cfg.TLSMinVersion())
	assert.Equal(t, uint16(0), cfg.TLSMaxVersion())
	assert.Empty(t, cfg.TLSServerName())
	assert.Empty(t, cfg.TLSRootCAFile())
	assert.Empty(t, cfg.TLSClientCertFile())
	assert.Empty(t, cfg.TLSClientKeyFile())

	require.Nil(t, cfg.RetryPolicy())
	require.Nil(t, cfg.CircuitBreaker())
}

func TestConfigYaml_ExplicitValues(t *testing.T) {
	t.Parallel()

	readTimeout := 3 * time.Second
	writeTimeout := 4 * time.Second

	cfg := &ConfigYaml{
		BaseUrlValue:             "https://api.example.com",
		InsecureSkipVerifyValue:  true,
		TimeoutValue:             time.Second,
		BaseHeadersValue:         Headers{"X-Key": []string{"v"}},
		UseRequestIdValue:        true,
		NameValue:                "custom",
		MaxConnsPerHostValue:     7,
		MaxIdleConnDurationValue: time.Minute,
		ReadTimeoutValue:         &readTimeout,
		WriteTimeoutValue:        &writeTimeout,
		MaxConnWaitTimeoutValue:  5 * time.Second,
		RetryPolicyValue:         &retry.ConfigYaml{EnabledValue: true},
		ReadBufferSizeValue:      1024,
		WriteBufferSizeValue:     2048,
		MaxResponseBodySizeValue: 4096,
		MaxConnDurationValue:     time.Hour,
		StreamResponseBodyValue:  true,
		FollowRedirectsValue:     true,
		TLSMinVersionValue:       TLSVersion12,
		TLSMaxVersionValue:       TLSVersion13,
		TLSServerNameValue:       "example.com",
		TLSRootCAFileValue:       "/ca.pem",
		TLSClientCertFileValue:   "/cert.pem",
		TLSClientKeyFileValue:    "/key.pem",
		CircuitBreakerValue:      &circuit_breaker.ConfigYaml{EnabledValue: true},
	}

	assert.Equal(t, "https://api.example.com", cfg.BaseUrl())
	assert.True(t, cfg.InsecureSkipVerify())
	assert.Equal(t, time.Second, cfg.Timeout())
	assert.Equal(t, Headers{"X-Key": []string{"v"}}, cfg.BaseHeaders())
	assert.True(t, cfg.UseRequestId())
	assert.Equal(t, "custom", cfg.Name())
	assert.Equal(t, 7, cfg.MaxConnsPerHost())
	assert.Equal(t, time.Minute, cfg.MaxIdleConnDuration())
	assert.Equal(t, readTimeout, cfg.ReadTimeout())
	assert.Equal(t, writeTimeout, cfg.WriteTimeout())
	assert.Equal(t, 5*time.Second, cfg.MaxConnWaitTimeout())
	assert.Equal(t, 1024, cfg.ReadBufferSize())
	assert.Equal(t, 2048, cfg.WriteBufferSize())
	assert.Equal(t, 4096, cfg.MaxResponseBodySize())
	assert.Equal(t, time.Hour, cfg.MaxConnDuration())
	assert.True(t, cfg.StreamResponseBody())
	assert.True(t, cfg.FollowRedirects())
	assert.Equal(t, TLSVersion12, cfg.TLSMinVersion())
	assert.Equal(t, TLSVersion13, cfg.TLSMaxVersion())
	assert.Equal(t, "example.com", cfg.TLSServerName())
	assert.Equal(t, "/ca.pem", cfg.TLSRootCAFile())
	assert.Equal(t, "/cert.pem", cfg.TLSClientCertFile())
	assert.Equal(t, "/key.pem", cfg.TLSClientKeyFile())

	require.NotNil(t, cfg.RetryPolicy())
	assert.True(t, cfg.RetryPolicy().Enabled())
	require.NotNil(t, cfg.CircuitBreaker())
	assert.True(t, cfg.CircuitBreaker().Enabled())
}
