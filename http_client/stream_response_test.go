package http_client

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pixality-inc/golang-core/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClientImpl_GetStream_Success(t *testing.T) {
	t.Parallel()

	expectedBody := "streaming response body content"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/stream", r.URL.Path)

		w.Header().Set("Content-Type", "application/octet-stream")
		w.WriteHeader(http.StatusOK)

		if _, err := w.Write([]byte(expectedBody)); err != nil {
			t.Errorf("failed to write response: %v", err)
		}
	}))
	defer server.Close()

	config := newTestConfig(server.URL)
	log := logger.NewLoggableImplWithService("test")
	client, err := NewClientImpl(log, config)
	require.NoError(t, err)

	resp, err := client.GetStream(context.Background(), "/stream")
	require.NoError(t, err)
	require.NotNil(t, resp)

	assert.Equal(t, http.StatusOK, resp.GetStatusCode())

	stream := resp.GetBody()
	defer stream.Close()

	body, err := io.ReadAll(stream)
	require.NoError(t, err)
	assert.Equal(t, expectedBody, string(body))
}

func TestClientImpl_GetStream_NotFound(t *testing.T) {
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
	client, err := NewClientImpl(log, config)
	require.NoError(t, err)

	resp, err := client.GetStream(context.Background(), "/missing")
	require.ErrorIs(t, err, ErrNotFound)
	require.Nil(t, resp)
}

func TestClientImpl_GetStream_BadRequest(t *testing.T) {
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
	client, err := NewClientImpl(log, config)
	require.NoError(t, err)

	resp, err := client.GetStream(context.Background(), "/bad")
	require.ErrorIs(t, err, ErrBadRequest)
	require.Nil(t, resp)
}

func TestClientImpl_GetStream_ServerError(t *testing.T) {
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
	client, err := NewClientImpl(log, config)
	require.NoError(t, err)

	resp, err := client.GetStream(context.Background(), "/error")
	require.ErrorIs(t, err, ErrNon200HttpCode)
	require.Nil(t, resp)
}

func TestClientImpl_GetStream_ContextCanceled(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := newTestConfig(server.URL)
	log := logger.NewLoggableImplWithService("test")
	client, err := NewClientImpl(log, config)
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	resp, err := client.GetStream(ctx, "/test")
	require.Error(t, err)
	require.Nil(t, resp)
}

func TestClientImpl_GetStream_BodyCloseDoesNotPanic(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)

		if _, err := w.Write([]byte("data")); err != nil {
			t.Errorf("failed to write response: %v", err)
		}
	}))
	defer server.Close()

	config := newTestConfig(server.URL)
	log := logger.NewLoggableImplWithService("test")
	client, err := NewClientImpl(log, config)
	require.NoError(t, err)

	resp, err := client.GetStream(context.Background(), "/test")
	require.NoError(t, err)

	// Read and close — should not panic
	_, err = io.ReadAll(resp.GetBody())
	require.NoError(t, err)

	require.NotPanics(t, func() {
		_ = resp.GetBody().Close()
	})
}
