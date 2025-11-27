package http_client

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/pixality-inc/golang-core/circuit_breaker"
	"github.com/pixality-inc/golang-core/logger"
	"github.com/pixality-inc/golang-core/retry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// test config implementation
type testConfig struct {
	baseUrl            string
	timeout            time.Duration
	insecureSkipVerify bool
	baseHeaders        Headers
	useRequestId       bool
	RetryPolicyValue   retry.Policy
}

// testConfigWithCB is testConfig with circuit breaker config
type testConfigWithCB struct {
	*testConfig

	cbConfig circuit_breaker.Config
}

func (c *testConfigWithCB) CircuitBreaker() circuit_breaker.Config {
	return c.cbConfig
}

func (c *testConfig) BaseUrl() string                        { return c.baseUrl }
func (c *testConfig) Timeout() time.Duration                 { return c.timeout }
func (c *testConfig) InsecureSkipVerify() bool               { return c.insecureSkipVerify }
func (c *testConfig) BaseHeaders() Headers                   { return c.baseHeaders }
func (c *testConfig) UseRequestId() bool                     { return c.useRequestId }
func (c *testConfig) Name() string                           { return "test_client" }
func (c *testConfig) MaxConnsPerHost() int                   { return DefaultMaxConnsPerHost }
func (c *testConfig) MaxIdleConnDuration() time.Duration     { return DefaultMaxIdleConnDuration }
func (c *testConfig) ReadTimeout() time.Duration             { return c.timeout }
func (c *testConfig) WriteTimeout() time.Duration            { return c.timeout }
func (c *testConfig) MaxConnWaitTimeout() time.Duration      { return 0 }
func (c *testConfig) RetryPolicy() retry.Policy              { return c.RetryPolicyValue }
func (c *testConfig) ReadBufferSize() int                    { return DefaultReadBufferSize }
func (c *testConfig) WriteBufferSize() int                   { return DefaultWriteBufferSize }
func (c *testConfig) MaxResponseBodySize() int               { return DefaultMaxResponseBodySize }
func (c *testConfig) MaxConnDuration() time.Duration         { return DefaultMaxConnDuration }
func (c *testConfig) StreamResponseBody() bool               { return false }
func (c *testConfig) TLSMinVersion() uint16                  { return 0 }
func (c *testConfig) TLSMaxVersion() uint16                  { return 0 }
func (c *testConfig) TLSServerName() string                  { return "" }
func (c *testConfig) TLSRootCAFile() string                  { return "" }
func (c *testConfig) TLSClientCertFile() string              { return "" }
func (c *testConfig) TLSClientKeyFile() string               { return "" }
func (c *testConfig) CircuitBreaker() circuit_breaker.Config { return nil }

func newTestConfig(baseUrl string) *testConfig {
	return &testConfig{
		baseUrl:     baseUrl,
		timeout:     10 * time.Second,
		baseHeaders: make(Headers),
	}
}

func TestClientImpl_Get(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/test", r.URL.Path)
		assert.Equal(t, "bar", r.URL.Query().Get("foo"))

		w.WriteHeader(http.StatusOK)

		if _, err := w.Write([]byte(`{"status":"ok"}`)); err != nil {
			t.Errorf("failed to write response: %v", err)
		}
	}))
	defer server.Close()

	config := newTestConfig(server.URL)
	log := logger.NewLoggableImplWithService("test")
	client, err := NewClientImpl(log, config, nil)
	require.NoError(t, err)

	resp, err := client.Get(context.Background(), "/test",
		WithQueryParam("foo", "bar"))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.GetStatusCode())
	assert.JSONEq(t, `{"status":"ok"}`, string(resp.GetBody()))
}

func TestClientImpl_Post(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/test", r.URL.Path)

		body, err := io.ReadAll(r.Body)
		assert.NoError(t, err)
		assert.JSONEq(t, `{"name":"test"}`, string(body))

		w.WriteHeader(http.StatusCreated)

		if _, err := w.Write([]byte(`{"id":123}`)); err != nil {
			t.Errorf("failed to write response: %v", err)
		}
	}))
	defer server.Close()

	config := newTestConfig(server.URL)
	log := logger.NewLoggableImplWithService("test")
	client, err := NewClientImpl(log, config, nil)
	require.NoError(t, err)

	resp, err := client.Post(context.Background(), "/test",
		WithBody([]byte(`{"name":"test"}`)))
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.GetStatusCode())
	assert.Equal(t, `{"id":123}`, string(resp.GetBody()))
}

func TestClientImpl_PostMultipart(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.True(t, strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data"))

		err := r.ParseMultipartForm(10 << 20)
		assert.NoError(t, err)

		assert.Equal(t, "test_value", r.FormValue("test_field"))

		w.WriteHeader(http.StatusOK)

		if _, err := w.Write([]byte(`{"uploaded":true}`)); err != nil {
			t.Errorf("failed to write response: %v", err)
		}
	}))
	defer server.Close()

	config := newTestConfig(server.URL)
	log := logger.NewLoggableImplWithService("test")
	client, err := NewClientImpl(log, config, nil)
	require.NoError(t, err)

	formData := NewFormDataImpl()
	err = formData.AddField("test_field", "test_value")
	require.NoError(t, err)

	resp, err := client.Post(context.Background(), "/test",
		WithFormData(formData))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.GetStatusCode())
	assert.Equal(t, `{"uploaded":true}`, string(resp.GetBody()))
}

func TestClientImpl_ErrorHandling_NotFound(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)

		if _, err := w.Write([]byte("not found")); err != nil {
			t.Errorf("failed to write response: %v", err)
		}
	}))
	defer server.Close()

	config := newTestConfig(server.URL)
	log := logger.NewLoggableImplWithService("test")
	client, err := NewClientImpl(log, config, nil)
	require.NoError(t, err)

	resp, err := client.Get(context.Background(), "/test", nil)
	require.ErrorIs(t, err, ErrNotFound)
	assert.Equal(t, http.StatusNotFound, resp.GetStatusCode())
}

func TestClientImpl_ErrorHandling_BadRequest(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)

		if _, err := w.Write([]byte("bad request")); err != nil {
			t.Errorf("failed to write response: %v", err)
		}
	}))
	defer server.Close()

	config := newTestConfig(server.URL)
	log := logger.NewLoggableImplWithService("test")
	client, err := NewClientImpl(log, config, nil)
	require.NoError(t, err)

	resp, err := client.Get(context.Background(), "/test", nil)
	require.ErrorIs(t, err, ErrBadRequest)
	assert.Equal(t, http.StatusBadRequest, resp.GetStatusCode())
}

func TestClientImpl_ErrorHandling_ServerError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)

		if _, err := w.Write([]byte("server error")); err != nil {
			t.Errorf("failed to write response: %v", err)
		}
	}))
	defer server.Close()

	config := newTestConfig(server.URL)
	log := logger.NewLoggableImplWithService("test")
	client, err := NewClientImpl(log, config, nil)
	require.NoError(t, err)

	resp, err := client.Get(context.Background(), "/test", nil)
	require.ErrorIs(t, err, ErrNon200HttpCode)
	assert.Equal(t, http.StatusInternalServerError, resp.GetStatusCode())
}

func TestClientImpl_Headers(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "test-value", r.Header.Get("X-Custom-Header"))
		assert.Equal(t, "base-value", r.Header.Get("X-Base-Header"))

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := newTestConfig(server.URL)
	config.baseHeaders = Headers{
		"X-Base-Header": []string{"base-value"},
	}

	log := logger.NewLoggableImplWithService("test")
	client, err := NewClientImpl(log, config, nil)
	require.NoError(t, err)

	_, err = client.Get(context.Background(), "/test",
		WithHeader("X-Custom-Header", "test-value"))
	require.NoError(t, err)
}

func TestClientImpl_Do(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PROPFIND", r.Method)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := newTestConfig(server.URL)
	log := logger.NewLoggableImplWithService("test")
	client, err := NewClientImpl(log, config, nil)
	require.NoError(t, err)

	resp, err := client.Do(context.Background(), "PROPFIND", "/test")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.GetStatusCode())
}

func TestFormData_AddField(t *testing.T) {
	t.Parallel()

	formData := NewFormDataImpl()
	err := formData.AddField("name", "value")
	require.NoError(t, err)
}

func TestFormData_AddFields(t *testing.T) {
	t.Parallel()

	formData := NewFormDataImpl()
	fields := FormFields{
		"field1": "value1",
		"field2": "value2",
	}
	err := formData.AddFields(fields)
	require.NoError(t, err)
}

func TestFormData_AddFile(t *testing.T) {
	t.Parallel()

	formData := NewFormDataImpl()
	body := bytes.NewBufferString("file content")
	err := formData.AddFile("file", "test.txt", "text/plain", body)
	require.NoError(t, err)
}

func TestAsJson(t *testing.T) {
	t.Parallel()

	type testEntity struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	response := &ResponseImpl{
		StatusCode: 200,
		Headers:    make(Headers),
		Body:       []byte(`{"name":"John","age":30}`),
	}

	typed, err := AsJson(response, testEntity{})
	require.NoError(t, err)
	assert.Equal(t, "John", typed.GetEntity().Name)
	assert.Equal(t, 30, typed.GetEntity().Age)
	assert.Equal(t, 200, typed.GetStatusCode())
}

func TestAsJson_InvalidJson(t *testing.T) {
	t.Parallel()

	type testEntity struct {
		Name string `json:"name"`
	}

	response := &ResponseImpl{
		StatusCode: 200,
		Headers:    make(Headers),
		Body:       []byte(`invalid json`),
	}

	_, err := AsJson(response, testEntity{})
	require.Error(t, err)
}

func TestNewClientImpl_WithCircuitBreakerOption(t *testing.T) {
	t.Parallel()

	log := logger.NewLoggableImplWithService("test")
	config := newTestConfig("")

	// custom circuit breaker
	customCB := circuit_breaker.New(&circuit_breaker.ConfigYaml{
		EnabledValue:             true,
		NameValue:                "custom_cb",
		ConsecutiveFailuresValue: 3,
	}, nil)

	// create client with custom CB via option
	client, err := NewClientImpl(log, config, nil, WithCircuitBreaker(customCB))
	require.NoError(t, err)
	require.NotNil(t, client)
	require.NotNil(t, client.circuitBreaker)
	assert.Equal(t, customCB, client.circuitBreaker)
}

func TestNewClientImpl_WithCircuitBreakerFromConfig(t *testing.T) {
	t.Parallel()

	log := logger.NewLoggableImplWithService("test")

	// config with circuit breaker settings
	config := &testConfig{
		timeout: 5 * time.Second,
	}

	cbConfig := &circuit_breaker.ConfigYaml{
		EnabledValue:             true,
		NameValue:                "http_test",
		ConsecutiveFailuresValue: 5,
	}

	configWithCB := &testConfigWithCB{
		testConfig: config,
		cbConfig:   cbConfig,
	}

	// create client - CB should be created automatically from config
	client, err := NewClientImpl(log, configWithCB, nil)
	require.NoError(t, err)
	require.NotNil(t, client)
	require.NotNil(t, client.circuitBreaker, "circuit breaker should be created from config")
}
