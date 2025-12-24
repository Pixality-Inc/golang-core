package http_test

import (
	"context"
	"testing"
	"time"

	"github.com/pixality-inc/golang-core/http"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

const address = "127.0.0.1:18080"

type testConfig struct{}

func (t *testConfig) Address() string {
	return address
}

func (t *testConfig) ShutdownTimeout() time.Duration {
	return time.Second
}

func testHandler(ctx *fasthttp.RequestCtx) {
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBodyString("ok")
}

func TestServer_ListenAndServe(t *testing.T) {
	t.Parallel()

	cfg := &testConfig{}

	srv := http.New("test", cfg, testHandler)

	go func() {
		err := srv.ListenAndServe(t.Context())
		if err != nil {
			t.Errorf("server error: %v", err)
		}
	}()

	time.Sleep(100 * time.Millisecond)

	client := fasthttp.Client{}
	req := fasthttp.AcquireRequest()

	resp := fasthttp.AcquireResponse()

	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	req.SetRequestURI("http://" + address)
	req.Header.SetMethod("GET")

	err := client.Do(req, resp)
	require.NoError(t, err)
	require.Equal(t, fasthttp.StatusOK, resp.StatusCode())
	require.Equal(t, "ok", string(resp.Body()))
}

func TestServer_Stop(t *testing.T) {
	t.Parallel()

	cfg := &testConfig{}

	srv := http.New("test", cfg, testHandler)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		_ = srv.ListenAndServe(ctx) // nolint:errcheck
	}()

	time.Sleep(100 * time.Millisecond)

	cancel()

	err := srv.Stop()
	require.NoError(t, err)
}

func TestServer_Name(t *testing.T) {
	t.Parallel()

	cfg := &testConfig{}
	srv := http.New("test-server", cfg, testHandler)
	require.Equal(t, "test-server", srv.Name())
}
