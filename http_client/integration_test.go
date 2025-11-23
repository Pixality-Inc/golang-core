package http_client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/pixality-inc/golang-core/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientImpl_Put(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := newTestConfig(server.URL)
	log := logger.NewLoggableImplWithService("test")
	client := NewClientImpl(log, config)

	resp, err := client.Put(context.Background(), "/test",
		WithBody([]byte(`{"update":"data"}`)))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestClientImpl_Patch(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := newTestConfig(server.URL)
	log := logger.NewLoggableImplWithService("test")
	client := NewClientImpl(log, config)

	resp, err := client.Patch(context.Background(), "/test",
		WithBody([]byte(`{"patch":"data"}`)))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestClientImpl_Delete(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	config := newTestConfig(server.URL)
	log := logger.NewLoggableImplWithService("test")
	client := NewClientImpl(log, config)

	resp, err := client.Delete(context.Background(), "/test")
	require.NoError(t, err)
	assert.Equal(t, http.StatusNoContent, resp.StatusCode)
}

func TestClientImpl_Head(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodHead, r.Method)
		w.Header().Set("Content-Length", "1234")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := newTestConfig(server.URL)
	log := logger.NewLoggableImplWithService("test")
	client := NewClientImpl(log, config)

	resp, err := client.Head(context.Background(), "/test")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestClientImpl_Options(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodOptions, r.Method)
		w.Header().Set("Allow", "GET, POST, PUT, DELETE")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := newTestConfig(server.URL)
	log := logger.NewLoggableImplWithService("test")
	client := NewClientImpl(log, config)

	resp, err := client.Options(context.Background(), "/test")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Headers["Allow"], "GET, POST, PUT, DELETE")
}

func TestClientImpl_WithJsonBody(t *testing.T) {
	type testData struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := newTestConfig(server.URL)
	log := logger.NewLoggableImplWithService("test")
	client := NewClientImpl(log, config)

	data := testData{Name: "John", Age: 30}
	resp, err := client.Post(context.Background(), "/test",
		WithJsonBody(data))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestClientImpl_MultipleOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "custom-value", r.Header.Get("X-Custom"))
		assert.Equal(t, "bar", r.URL.Query().Get("foo"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := newTestConfig(server.URL)
	log := logger.NewLoggableImplWithService("test")
	client := NewClientImpl(log, config)

	resp, err := client.Get(context.Background(), "/test",
		WithHeader("X-Custom", "custom-value"),
		WithQueryParam("foo", "bar"))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestClientImpl_RetryOnServerError(t *testing.T) {
	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := attempts.Add(1)
		if count < 3 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := newTestConfig(server.URL)
	config.RetryPolicyValue = &RetryPolicy{
		MaxAttempts:        3,
		InitialInterval:    10 * time.Millisecond,
		BackoffCoefficient: 2.0,
		MaxInterval:        100 * time.Millisecond,
	}

	log := logger.NewLoggableImplWithService("test")
	client := NewClientImpl(log, config)

	resp, err := client.Get(context.Background(), "/test")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, int32(3), attempts.Load())
}

func TestClientImpl_RetryExhaustion(t *testing.T) {
	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	config := newTestConfig(server.URL)
	config.RetryPolicyValue = &RetryPolicy{
		MaxAttempts:        3,
		InitialInterval:    10 * time.Millisecond,
		BackoffCoefficient: 2.0,
		MaxInterval:        100 * time.Millisecond,
	}

	log := logger.NewLoggableImplWithService("test")
	client := NewClientImpl(log, config)

	resp, err := client.Get(context.Background(), "/test")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNon200HttpCode)
	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)
	assert.Equal(t, int32(3), attempts.Load())
}

func TestClientImpl_RetryContextCancellation(t *testing.T) {
	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		time.Sleep(50 * time.Millisecond)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	config := newTestConfig(server.URL)
	config.RetryPolicyValue = &RetryPolicy{
		MaxAttempts:        10,
		InitialInterval:    10 * time.Millisecond,
		BackoffCoefficient: 2.0,
		MaxInterval:        100 * time.Millisecond,
	}

	log := logger.NewLoggableImplWithService("test")
	client := NewClientImpl(log, config)

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()

	_, err := client.Get(ctx, "/test")
	require.Error(t, err)
	// should not exhaust all 10 attempts due to context timeout
	assert.Less(t, attempts.Load(), int32(10))
}

func TestClientImpl_NoRetryOn4xx(t *testing.T) {
	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	config := newTestConfig(server.URL)
	config.RetryPolicyValue = &RetryPolicy{
		MaxAttempts:        3,
		InitialInterval:    10 * time.Millisecond,
		BackoffCoefficient: 2.0,
		MaxInterval:        100 * time.Millisecond,
	}

	log := logger.NewLoggableImplWithService("test")
	client := NewClientImpl(log, config)

	_, err := client.Get(context.Background(), "/test")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrNotFound)
	// should only attempt once (no retry on 404)
	assert.Equal(t, int32(1), attempts.Load())
}

func TestClientImpl_RetryOn429(t *testing.T) {
	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count := attempts.Add(1)
		if count < 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := newTestConfig(server.URL)
	config.RetryPolicyValue = &RetryPolicy{
		MaxAttempts:        3,
		InitialInterval:    10 * time.Millisecond,
		BackoffCoefficient: 2.0,
		MaxInterval:        100 * time.Millisecond,
	}

	log := logger.NewLoggableImplWithService("test")
	client := NewClientImpl(log, config)

	resp, err := client.Get(context.Background(), "/test")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, int32(2), attempts.Load())
}
