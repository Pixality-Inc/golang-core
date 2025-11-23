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

	"github.com/pixality-inc/golang-core/logger"
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
	RetryPolicyValue   *RetryPolicy
}

func (c *testConfig) BaseUrl() string                    { return c.baseUrl }
func (c *testConfig) Timeout() time.Duration             { return c.timeout }
func (c *testConfig) InsecureSkipVerify() bool           { return c.insecureSkipVerify }
func (c *testConfig) BaseHeaders() Headers               { return c.baseHeaders }
func (c *testConfig) UseRequestId() bool                 { return c.useRequestId }
func (c *testConfig) Name() string                       { return "test_client" }
func (c *testConfig) MaxConnsPerHost() int               { return DefaultMaxConnsPerHost }
func (c *testConfig) MaxIdleConnDuration() time.Duration { return DefaultMaxIdleConnDuration }
func (c *testConfig) ReadTimeout() time.Duration         { return c.timeout }
func (c *testConfig) WriteTimeout() time.Duration        { return c.timeout }
func (c *testConfig) MaxConnWaitTimeout() time.Duration  { return 0 }
func (c *testConfig) RetryPolicy() *RetryPolicy          { return c.RetryPolicyValue }

func newTestConfig(baseUrl string) *testConfig {
	return &testConfig{
		baseUrl:     baseUrl,
		timeout:     10 * time.Second,
		baseHeaders: make(Headers),
	}
}

func TestClientImpl_Get(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/test", r.URL.Path)
		assert.Equal(t, "bar", r.URL.Query().Get("foo"))

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	config := newTestConfig(server.URL)
	log := logger.NewLoggableImplWithService("test")
	client := NewClientImpl(log, config)

	resp, err := client.Get(context.Background(), "/test",
		WithQueryParam("foo", "bar"))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, `{"status":"ok"}`, string(resp.Body))
}

func TestClientImpl_Post(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/test", r.URL.Path)

		body, _ := io.ReadAll(r.Body)
		assert.Equal(t, `{"name":"test"}`, string(body))

		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"id":123}`))
	}))
	defer server.Close()

	config := newTestConfig(server.URL)
	log := logger.NewLoggableImplWithService("test")
	client := NewClientImpl(log, config)

	resp, err := client.Post(context.Background(), "/test",
		WithBody([]byte(`{"name":"test"}`)))
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)
	assert.Equal(t, `{"id":123}`, string(resp.Body))
}

func TestClientImpl_PostMultipart(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.True(t, strings.HasPrefix(r.Header.Get("Content-Type"), "multipart/form-data"))

		err := r.ParseMultipartForm(10 << 20)
		require.NoError(t, err)

		assert.Equal(t, "test_value", r.FormValue("test_field"))

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"uploaded":true}`))
	}))
	defer server.Close()

	config := newTestConfig(server.URL)
	log := logger.NewLoggableImplWithService("test")
	client := NewClientImpl(log, config)

	formData := NewFormData()
	err := formData.AddField("test_field", "test_value")
	require.NoError(t, err)

	resp, err := client.Post(context.Background(), "/test",
		WithFormData(formData))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, `{"uploaded":true}`, string(resp.Body))
}

func TestClientImpl_ErrorHandling_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("not found"))
	}))
	defer server.Close()

	config := newTestConfig(server.URL)
	log := logger.NewLoggableImplWithService("test")
	client := NewClientImpl(log, config)

	resp, err := client.Get(context.Background(), "/test", nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotFound)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestClientImpl_ErrorHandling_BadRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("bad request"))
	}))
	defer server.Close()

	config := newTestConfig(server.URL)
	log := logger.NewLoggableImplWithService("test")
	client := NewClientImpl(log, config)

	resp, err := client.Get(context.Background(), "/test", nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrBadRequest)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestClientImpl_ErrorHandling_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server error"))
	}))
	defer server.Close()

	config := newTestConfig(server.URL)
	log := logger.NewLoggableImplWithService("test")
	client := NewClientImpl(log, config)

	resp, err := client.Get(context.Background(), "/test", nil)
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNon200HttpCode)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
}

func TestClientImpl_Headers(t *testing.T) {
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
	client := NewClientImpl(log, config)

	_, err := client.Get(context.Background(), "/test",
		WithHeader("X-Custom-Header", "test-value"))
	require.NoError(t, err)
}

func TestClientImpl_Do(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PROPFIND", r.Method)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := newTestConfig(server.URL)
	log := logger.NewLoggableImplWithService("test")
	client := NewClientImpl(log, config)

	resp, err := client.Do(context.Background(), "PROPFIND", "/test")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestFormData_AddField(t *testing.T) {
	formData := NewFormData()
	err := formData.AddField("name", "value")
	require.NoError(t, err)
}

func TestFormData_AddFields(t *testing.T) {
	formData := NewFormData()
	fields := FormFields{
		"field1": "value1",
		"field2": "value2",
	}
	err := formData.AddFields(fields)
	require.NoError(t, err)
}

func TestFormData_AddFile(t *testing.T) {
	formData := NewFormData()
	body := bytes.NewBuffer([]byte("file content"))
	err := formData.AddFile("file", "test.txt", "text/plain", body)
	require.NoError(t, err)
}

func TestAsJson(t *testing.T) {
	type testEntity struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	response := &Response{
		StatusCode: 200,
		Headers:    make(Headers),
		Body:       []byte(`{"name":"John","age":30}`),
	}

	typed, err := AsJson(response, testEntity{})
	require.NoError(t, err)
	assert.Equal(t, "John", typed.Entity.Name)
	assert.Equal(t, 30, typed.Entity.Age)
	assert.Equal(t, 200, typed.StatusCode)
}

func TestAsJson_InvalidJson(t *testing.T) {
	type testEntity struct {
		Name string `json:"name"`
	}

	response := &Response{
		StatusCode: 200,
		Headers:    make(Headers),
		Body:       []byte(`invalid json`),
	}

	_, err := AsJson(response, testEntity{})
	require.Error(t, err)
}
