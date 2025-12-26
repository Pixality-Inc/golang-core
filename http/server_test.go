package http_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/pixality-inc/golang-core/http"
	"github.com/stretchr/testify/require"
	"github.com/valyala/fasthttp"
)

type testConfig struct {
	addr string
}

func newTestConfig(t *testing.T) *testConfig {
	t.Helper()

	var lc net.ListenConfig

	ln, err := lc.Listen(context.Background(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)

	addr := ln.Addr().String()
	require.NoError(t, ln.Close())

	return &testConfig{
		addr: addr,
	}
}

func (c *testConfig) Address() string {
	return c.addr
}

func (t *testConfig) ShutdownTimeout() time.Duration {
	return time.Second
}

func waitServer(t *testing.T, addr string) {
	t.Helper()

	client := fasthttp.Client{}
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()

	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	req.SetRequestURI("http://" + addr)
	req.Header.SetMethod("GET")

	require.Eventually(t, func() bool {
		err := client.Do(req, resp)

		return err == nil
	}, time.Second, 10*time.Millisecond)
}

func testHandler(ctx *fasthttp.RequestCtx) {
	ctx.SetStatusCode(fasthttp.StatusOK)
	ctx.SetBodyString("ok")
}

func TestServer_ListenAndServe(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig(t)

	srv := http.New("test", cfg, testHandler)

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	go func() {
		err := srv.ListenAndServe(ctx)
		if err != nil {
			t.Errorf("server error: %v", err)
		}
	}()

	waitServer(t, cfg.Address())

	client := fasthttp.Client{}
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()

	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	req.SetRequestURI("http://" + cfg.Address())
	req.Header.SetMethod("GET")

	err := client.Do(req, resp)
	require.NoError(t, err)
	require.Equal(t, fasthttp.StatusOK, resp.StatusCode())
	require.Equal(t, "ok", string(resp.Body()))
}

func TestServer_Stop(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig(t)

	srv := http.New("test", cfg, testHandler)

	ctx, cancel := context.WithCancel(t.Context())

	go func() {
		err := srv.ListenAndServe(ctx)
		if err != nil {
			t.Errorf("server error: %v", err)
		}
	}()

	waitServer(t, cfg.Address())

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
