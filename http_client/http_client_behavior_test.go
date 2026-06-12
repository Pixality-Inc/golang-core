package http_client

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"

	"github.com/pixality-inc/golang-core/circuit_breaker"
	http2 "github.com/pixality-inc/golang-core/http"
	"github.com/pixality-inc/golang-core/logger"
	"github.com/pixality-inc/golang-core/retry"
)

var errConnRefused = errors.New("dial tcp: connection refused")

func newRetryPolicy(retryNonIdempotent bool) retry.Policy {
	return retry.NewPolicy(
		retry.WithEnabled(true),
		retry.WithMaxAttempts(3),
		retry.WithInitialInterval(10*time.Millisecond),
		retry.WithBackoffCoefficient(2.0),
		retry.WithMaxInterval(100*time.Millisecond),
		retry.WithRetryNonIdempotent(retryNonIdempotent),
	)
}

func newTestClient(t *testing.T, config Config) *ClientImpl {
	t.Helper()

	client, err := NewClientImpl(logger.NewLoggableImplWithService("test"), config)
	require.NoError(t, err)

	return client
}

func TestClientImpl_NoRetryNonIdempotentByDefault(t *testing.T) {
	t.Parallel()

	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	config := newTestConfig(server.URL)
	config.RetryPolicyValue = newRetryPolicy(false)

	client := newTestClient(t, config)

	_, err := client.Post(context.Background(), "/test", WithBody([]byte(`{}`)))
	require.ErrorIs(t, err, ErrNon200HttpCode)
	assert.Equal(t, int32(1), attempts.Load())
}

func TestClientImpl_RetryNonIdempotentWhenEnabled(t *testing.T) {
	t.Parallel()

	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempts.Add(1) < 3 {
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := newTestConfig(server.URL)
	config.RetryPolicyValue = newRetryPolicy(true)

	client := newTestClient(t, config)

	resp, err := client.Post(context.Background(), "/test", WithBody([]byte(`{}`)))
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.GetStatusCode())
	assert.Equal(t, int32(3), attempts.Load())
}

func TestClientImpl_RetryIdempotentDeleteByDefault(t *testing.T) {
	t.Parallel()

	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if attempts.Add(1) < 2 {
			w.WriteHeader(http.StatusInternalServerError)

			return
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := newTestConfig(server.URL)
	config.RetryPolicyValue = newRetryPolicy(false)

	client := newTestClient(t, config)

	resp, err := client.Delete(context.Background(), "/test")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.GetStatusCode())
	assert.Equal(t, int32(2), attempts.Load())
}

func TestClientImpl_Timeout(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(150 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := newTestConfig(server.URL)
	config.timeout = 50 * time.Millisecond

	client := newTestClient(t, config)

	_, err := client.Get(context.Background(), "/test")
	require.ErrorIs(t, err, fasthttp.ErrTimeout)
}

func TestClientImpl_ContextCancellation(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := newTestClient(t, newTestConfig(server.URL))

	ctx, cancel := context.WithCancel(context.Background())
	time.AfterFunc(50*time.Millisecond, cancel)

	_, err := client.Get(ctx, "/test")
	require.ErrorIs(t, err, context.Canceled)
}

func TestClientImpl_FollowRedirects(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/start" {
			http.Redirect(w, r, "/final", http.StatusFound)

			return
		}

		w.WriteHeader(http.StatusOK)

		_, err := w.Write([]byte("final"))
		assert.NoError(t, err)
	}))
	defer server.Close()

	config := newTestConfig(server.URL)
	config.followRedirects = true

	client := newTestClient(t, config)

	resp, err := client.Get(context.Background(), "/start")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.GetStatusCode())
	assert.Equal(t, "final", string(resp.GetBody()))
}

func TestClientImpl_NoFollowRedirects(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/final", http.StatusFound)
	}))
	defer server.Close()

	client := newTestClient(t, newTestConfig(server.URL))

	resp, err := client.Get(context.Background(), "/start")
	require.ErrorIs(t, err, ErrNon200HttpCode)
	assert.Equal(t, http.StatusFound, resp.GetStatusCode())
}

func TestClientImpl_RequestIdPropagation(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "test-request-id", r.Header.Get("X-Request-Id"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := newTestConfig(server.URL)
	config.useRequestId = true

	client := newTestClient(t, config)

	ctx := context.WithValue(context.Background(), http2.RequestIdValueKey, "test-request-id") //nolint:staticcheck

	_, err := client.Get(ctx, "/test")
	require.NoError(t, err)
}

func TestClientImpl_NoRequestIdWithoutContextValue(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.Header.Get("X-Request-Id"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := newTestConfig(server.URL)
	config.useRequestId = true

	client := newTestClient(t, config)

	_, err := client.Get(context.Background(), "/test")
	require.NoError(t, err)
}

func TestClientImpl_BothFormDataAndBody(t *testing.T) {
	t.Parallel()

	client := newTestClient(t, newTestConfig("http://localhost"))

	formData := NewFormDataImpl()
	require.NoError(t, formData.AddField("field", "value"))

	_, err := client.Post(context.Background(), "/test",
		WithFormData(formData),
		WithBody([]byte(`{}`)))
	require.ErrorIs(t, err, ErrBothFormDataAndBody)
}

func TestClientImpl_EmptyBaseUrl(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/absolute", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := newTestClient(t, newTestConfig(""))

	resp, err := client.Get(context.Background(), server.URL+"/absolute")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.GetStatusCode())
}

func TestClientImpl_CircuitBreakerOpensAfterFailures(t *testing.T) {
	t.Parallel()

	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	config := &testConfigWithCB{
		testConfig: newTestConfig(server.URL),
		cbConfig: &circuit_breaker.ConfigYaml{
			EnabledValue:             true,
			NameValue:                "test_cb",
			ConsecutiveFailuresValue: 3,
		},
	}

	client := newTestClient(t, config)

	for range 5 {
		_, err := client.Get(context.Background(), "/test")
		require.Error(t, err)
	}

	assert.Equal(t, int32(3), attempts.Load())
}

func TestClientImpl_CircuitBreakerIgnores4xx(t *testing.T) {
	t.Parallel()

	var attempts atomic.Int32

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts.Add(1)
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	config := &testConfigWithCB{
		testConfig: newTestConfig(server.URL),
		cbConfig: &circuit_breaker.ConfigYaml{
			EnabledValue:             true,
			NameValue:                "test_cb_4xx",
			ConsecutiveFailuresValue: 3,
		},
	}

	client := newTestClient(t, config)

	for range 5 {
		_, err := client.Get(context.Background(), "/test")
		require.ErrorIs(t, err, ErrNotFound)
	}

	assert.Equal(t, int32(5), attempts.Load())
}

func TestShouldIgnoreErrorForCircuitBreaker(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name     string
		err      error
		expected bool
	}{
		{"nil", nil, false},
		{"context canceled", context.Canceled, true},
		{"not found", ErrNotFound, true},
		{"bad request", ErrBadRequest, true},
		{"forbidden 403", fmt.Errorf("%w: %d", ErrNon200HttpCode, http.StatusForbidden), true},
		{"unauthorized 401", fmt.Errorf("%w: %d", ErrNon200HttpCode, http.StatusUnauthorized), true},
		{"server error 500", fmt.Errorf("%w: %d", ErrNon200HttpCode, http.StatusInternalServerError), false},
		{"bad gateway 502", fmt.Errorf("%w: %d", ErrNon200HttpCode, http.StatusBadGateway), false},
		{"net timeout", &net.DNSError{Err: "timeout", IsTimeout: true}, false},
		{"connection refused", errConnRefused, false},
		{"unknown error", errors.ErrUnsupported, false},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			assert.Equal(t, testCase.expected, ShouldIgnoreErrorForCircuitBreaker(testCase.err))
		})
	}
}

func TestClientImpl_WithHeadersAndQueryParams(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "value-1", r.Header.Get("X-First"))
		assert.Equal(t, "value-2", r.Header.Get("X-Second"))
		assert.Equal(t, "bar", r.URL.Query().Get("foo"))
		assert.Equal(t, "qux", r.URL.Query().Get("baz"))
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := newTestClient(t, newTestConfig(server.URL))

	_, err := client.Get(context.Background(), "/test",
		WithHeaders(Headers{
			"X-First":  []string{"value-1"},
			"X-Second": []string{"value-2"},
		}),
		WithQueryParams(QueryParams{
			"foo": "bar",
			"baz": "qux",
		}))
	require.NoError(t, err)
}

func TestFormData_BuildCaching(t *testing.T) {
	t.Parallel()

	formData := NewFormDataImpl()
	require.NoError(t, formData.AddField("field", "value"))

	firstBody, firstContentType, err := formData.Build()
	require.NoError(t, err)
	require.NotNil(t, firstBody)
	assert.Contains(t, firstContentType, "multipart/form-data")
	assert.Contains(t, firstBody.String(), "value")

	secondBody, secondContentType, err := formData.Build()
	require.NoError(t, err)
	assert.Same(t, firstBody, secondBody)
	assert.Equal(t, firstContentType, secondContentType)
}

func TestResponseImpl_DecodeJSON(t *testing.T) {
	t.Parallel()

	response := &ResponseImpl{
		StatusCode: http.StatusOK,
		Body:       []byte(`{"name":"John"}`),
	}

	var decoded struct {
		Name string `json:"name"`
	}

	require.NoError(t, response.DecodeJSON(&decoded))
	assert.Equal(t, "John", decoded.Name)

	response.Body = []byte("not json")
	require.Error(t, response.DecodeJSON(&decoded))
}

func TestResponseImpl_String(t *testing.T) {
	t.Parallel()

	response := &ResponseImpl{
		StatusCode: http.StatusTeapot,
		Body:       []byte("hello"),
	}

	assert.Equal(t, "StatusCode: 418, Body: hello", response.String())
}

func generateTestCert(t *testing.T) (certPEM, keyPEM []byte) {
	t.Helper()

	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	require.NoError(t, err)

	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}

	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	require.NoError(t, err)

	keyDER, err := x509.MarshalECPrivateKey(key)
	require.NoError(t, err)

	certPEM = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
	keyPEM = pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER})

	return certPEM, keyPEM
}

func writeTempFile(t *testing.T, name string, content []byte) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	require.NoError(t, os.WriteFile(path, content, 0o600))

	return path
}

func TestNewClientImpl_TLSRootCA(t *testing.T) {
	t.Parallel()

	certPEM, _ := generateTestCert(t)

	config := newTestConfig("")
	config.tlsRootCAFile = writeTempFile(t, "ca.pem", certPEM)

	client := newTestClient(t, config)
	require.NotNil(t, client.client.TLSConfig)
	assert.NotNil(t, client.client.TLSConfig.RootCAs)
}

func TestNewClientImpl_TLSRootCAFileNotFound(t *testing.T) {
	t.Parallel()

	config := newTestConfig("")
	config.tlsRootCAFile = "/nonexistent/ca.pem"

	_, err := NewClientImpl(logger.NewLoggableImplWithService("test"), config)
	require.ErrorIs(t, err, ErrTLSConfig)
}

func TestNewClientImpl_TLSRootCAInvalidPEM(t *testing.T) {
	t.Parallel()

	config := newTestConfig("")
	config.tlsRootCAFile = writeTempFile(t, "ca.pem", []byte("not a pem"))

	_, err := NewClientImpl(logger.NewLoggableImplWithService("test"), config)
	require.ErrorIs(t, err, ErrTLSConfig)
}

func TestNewClientImpl_TLSClientCertificate(t *testing.T) {
	t.Parallel()

	certPEM, keyPEM := generateTestCert(t)

	config := newTestConfig("")
	config.tlsClientCertFile = writeTempFile(t, "cert.pem", certPEM)
	config.tlsClientKeyFile = writeTempFile(t, "key.pem", keyPEM)

	client := newTestClient(t, config)
	require.NotNil(t, client.client.TLSConfig)
	assert.Len(t, client.client.TLSConfig.Certificates, 1)
}

func TestNewClientImpl_TLSClientCertificateInvalid(t *testing.T) {
	t.Parallel()

	config := newTestConfig("")
	config.tlsClientCertFile = writeTempFile(t, "cert.pem", []byte("bad cert"))
	config.tlsClientKeyFile = writeTempFile(t, "key.pem", []byte("bad key"))

	_, err := NewClientImpl(logger.NewLoggableImplWithService("test"), config)
	require.ErrorIs(t, err, ErrTLSConfig)
}
