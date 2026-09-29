package pushwoosh

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/pixality-inc/golang-core/pusher"
)

const (
	testDeviceApiKey = "device-key"
	testServerApiKey = "server-key"
)

type recordedRequest struct {
	path          string
	authorization string
}

type testServer struct {
	mu       sync.Mutex
	requests []recordedRequest
}

func (s *testServer) recorded() []recordedRequest {
	s.mu.Lock()
	defer s.mu.Unlock()

	return append([]recordedRequest(nil), s.requests...)
}

func newTestClient(t *testing.T, handler http.HandlerFunc) (Client, *testServer) {
	t.Helper()

	recorder := &testServer{}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		recorder.mu.Lock()
		recorder.requests = append(recorder.requests, recordedRequest{
			path:          r.URL.Path,
			authorization: r.Header.Get("Authorization"),
		})
		recorder.mu.Unlock()

		handler(w, r)
	}))
	t.Cleanup(server.Close)

	client, err := NewClient(NewClientConfig(server.URL, "app", testDeviceApiKey, testServerApiKey, time.Second))
	require.NoError(t, err)

	return client, recorder
}

func respondJson(body string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		// a failed write surfaces in the test as the client error it causes
		_, _ = w.Write([]byte(body)) //nolint:errcheck
	}
}

// respondRaw writes raw bytes to the connection and closes it, for responses net/http cannot produce
func respondRaw(raw string) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		hijacker, ok := w.(http.Hijacker)
		if !ok {
			return
		}

		conn, _, err := hijacker.Hijack()
		if err != nil {
			return
		}

		_, _ = conn.Write([]byte(raw)) //nolint:errcheck

		_ = conn.Close()
	}
}

func respondStatus(statusCode int) http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(statusCode)
	}
}

func TestClient_AuthorizationPerApiMethod(t *testing.T) {
	t.Parallel()

	client, recorder := newTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/messaging/v2/notify" {
			respondJson(`{"result":{"message_code":"msg-1"}}`)(w, r)

			return
		}

		respondJson(`{"status_code":200,"status_message":"OK","response":{}}`)(w, r)
	})

	ctx := context.Background()

	require.NoError(t, client.RegisterDevice(ctx, DeviceTypeIOS, "user", "device", "token"))
	require.NoError(t, client.UnregisterDevice(ctx, "device"))

	result, err := client.Notify(ctx, AllPlatformTypes, MessageTypeTransactional, MessagePayload{}, WithSendAt(time.Now()))
	require.NoError(t, err)
	require.Equal(t, "msg-1", result.MessageId)

	require.Equal(t, []recordedRequest{
		{path: "/json/1.3/registerDevice", authorization: "Token " + testDeviceApiKey},
		{path: "/json/1.3/unregisterDevice", authorization: "Token " + testDeviceApiKey},
		{path: "/messaging/v2/notify", authorization: "Token " + testServerApiKey},
	}, recorder.recorded())
}

func TestClient_DeviceApiErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		handler       http.HandlerFunc
		wantTemporary bool
		wantMessage   string
	}{
		{
			name:          "http 5xx is temporary",
			handler:       respondStatus(http.StatusBadGateway),
			wantTemporary: true,
		},
		{
			name:          "http 429 is temporary",
			handler:       respondStatus(http.StatusTooManyRequests),
			wantTemporary: true,
		},
		{
			name:          "http 408 is temporary",
			handler:       respondStatus(http.StatusRequestTimeout),
			wantTemporary: true,
		},
		{
			name:          "http 400 is permanent",
			handler:       respondStatus(http.StatusBadRequest),
			wantTemporary: false,
		},
		{
			name:          "http 404 is permanent",
			handler:       respondStatus(http.StatusNotFound),
			wantTemporary: false,
		},
		{
			name:          "http 401 is permanent",
			handler:       respondStatus(http.StatusUnauthorized),
			wantTemporary: false,
		},
		{
			name:          "api status 500 in the body is temporary",
			handler:       respondJson(`{"status_code":500,"status_message":"Internal Error","response":{}}`),
			wantTemporary: true,
			wantMessage:   "failed to register device: Internal Error",
		},
		{
			name:          "api argument error in the body is permanent",
			handler:       respondJson(`{"status_code":210,"status_message":"Argument error","response":{}}`),
			wantTemporary: false,
			wantMessage:   "failed to register device: Argument error",
		},
		{
			name:          "null body is permanent",
			handler:       respondJson(`null`),
			wantTemporary: false,
		},
		{
			name:          "response cut inside a chunked body is temporary",
			handler:       respondRaw("HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nTransfer-Encoding: chunked\r\n\r\n5\r\n{\"sta"),
			wantTemporary: true,
		},
		{
			name: "connection closed before the response is temporary",
			handler: func(w http.ResponseWriter, _ *http.Request) {
				hijacker, ok := w.(http.Hijacker)
				if !ok {
					return
				}

				conn, _, err := hijacker.Hijack()
				if err == nil {
					_ = conn.Close()
				}
			},
			wantTemporary: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			client, _ := newTestClient(t, testCase.handler)

			err := client.RegisterDevice(context.Background(), DeviceTypeIOS, "user", "device", "token")
			require.ErrorIs(t, err, ErrRegisterDevice)
			require.Equal(t, testCase.wantTemporary, isTemporary(err), err.Error())

			if testCase.wantMessage != "" {
				require.Contains(t, err.Error(), testCase.wantMessage)
			}
		})
	}
}

// newSilentListener accepts connections and never answers, neither http nor a tls handshake
func newSilentListener(t *testing.T) string {
	t.Helper()

	var listenConfig net.ListenConfig

	listener, err := listenConfig.Listen(context.Background(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)

	var (
		connsMu sync.Mutex
		conns   []net.Conn
		closed  bool
	)

	t.Cleanup(func() {
		_ = listener.Close()

		connsMu.Lock()
		defer connsMu.Unlock()

		closed = true

		for _, conn := range conns {
			_ = conn.Close()
		}
	})

	// accepts connections and never answers
	go func() {
		for {
			conn, acceptErr := listener.Accept()
			if acceptErr != nil {
				return
			}

			connsMu.Lock()

			if closed {
				_ = conn.Close()
			} else {
				conns = append(conns, conn)
			}

			connsMu.Unlock()
		}
	}()

	return listener.Addr().String()
}

func TestClient_TimeoutIsTemporary(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		scheme string
	}{
		{name: "silent http server", scheme: "http"},
		// the handshake runs lazily inside the request write, so without a write timeout it would
		// wait for the server hello forever
		{name: "silent tls handshake", scheme: "https"},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			baseUrl := testCase.scheme + "://" + newSilentListener(t)

			client, err := NewClient(NewClientConfig(baseUrl, "app", testDeviceApiKey, testServerApiKey, 100*time.Millisecond))
			require.NoError(t, err)

			done := make(chan error, 1)

			go func() {
				done <- client.UnregisterDevice(context.Background(), "device")
			}()

			select {
			case err = <-done:
			case <-time.After(5 * time.Second):
				t.Fatal("the request outlived its timeout")
			}

			require.ErrorIs(t, err, ErrUnregisterDevice)
			require.True(t, isTemporary(err), err.Error())
		})
	}
}

func TestClient_CanceledContextIsPermanent(t *testing.T) {
	t.Parallel()

	client, _ := newTestClient(t, respondJson(`{"status_code":200,"status_message":"OK","response":{}}`))

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := client.UnregisterDevice(ctx, "device")
	require.ErrorIs(t, err, context.Canceled)
	require.False(t, isTemporary(err))
}

func isTemporary(err error) bool {
	return errors.Is(err, pusher.ErrTemporary)
}

func TestClient_NotifyErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		handler       http.HandlerFunc
		wantTemporary bool
		wantErr       error
	}{
		{name: "http 5xx is temporary", handler: respondStatus(http.StatusServiceUnavailable), wantTemporary: true},
		{name: "http 429 is temporary", handler: respondStatus(http.StatusTooManyRequests), wantTemporary: true},
		{name: "http 401 is permanent", handler: respondStatus(http.StatusUnauthorized), wantTemporary: false},
		{name: "null body is permanent", handler: respondJson(`null`), wantTemporary: false, wantErr: ErrEmptyResponse},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			client, _ := newTestClient(t, testCase.handler)

			_, err := client.Notify(context.Background(), AllPlatformTypes, MessageTypeTransactional, MessagePayload{}, WithSendAt(time.Now()))
			require.ErrorIs(t, err, ErrNotify)
			require.Equal(t, testCase.wantTemporary, isTemporary(err), err.Error())

			if testCase.wantErr != nil {
				require.ErrorIs(t, err, testCase.wantErr)
			}
		})
	}
}

func TestClient_ExpiredContextIsPermanent(t *testing.T) {
	t.Parallel()

	client, _ := newTestClient(t, respondJson(`{"status_code":200,"status_message":"OK","response":{}}`))

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(-time.Second))
	defer cancel()

	err := client.RegisterDevice(ctx, DeviceTypeIOS, "user", "device", "token")
	require.ErrorIs(t, err, context.DeadlineExceeded)
	require.False(t, isTemporary(err))
}

func TestClient_TransportErrors(t *testing.T) {
	t.Parallel()

	var listenConfig net.ListenConfig

	// a port nobody listens on: bound and released right away
	listener, err := listenConfig.Listen(context.Background(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)

	refusedUrl := "http://" + listener.Addr().String()
	require.NoError(t, listener.Close())

	tests := []struct {
		name          string
		baseUrl       string
		wantTemporary bool
	}{
		{name: "refused connection is temporary", baseUrl: refusedUrl, wantTemporary: true},
		{name: "unsupported scheme is permanent", baseUrl: "ftp://127.0.0.1", wantTemporary: false},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			client, err := NewClient(NewClientConfig(testCase.baseUrl, "app", testDeviceApiKey, testServerApiKey, time.Second))
			require.NoError(t, err)

			err = client.UnregisterDevice(context.Background(), "device")
			require.ErrorIs(t, err, ErrUnregisterDevice)
			require.Equal(t, testCase.wantTemporary, isTemporary(err), err.Error())
		})
	}
}

// fatal handshake_failure alert record: content type 21, tls 1.2, length 2, level fatal, code 40
var tlsHandshakeFailureAlert = []byte{0x15, 0x03, 0x03, 0x00, 0x02, 0x02, 0x28}

func TestClient_TlsAlertOfServerIsPermanent(t *testing.T) {
	t.Parallel()

	var listenConfig net.ListenConfig

	listener, err := listenConfig.Listen(context.Background(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)

	t.Cleanup(func() { _ = listener.Close() })

	// answers the client hello with an alert, the way a server refusing the tls setup does
	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr != nil {
			return
		}

		defer func() { _ = conn.Close() }()

		clientHello := make([]byte, 1024)
		if _, readErr := conn.Read(clientHello); readErr != nil {
			return
		}

		_, _ = conn.Write(tlsHandshakeFailureAlert) //nolint:errcheck
	}()

	client, err := NewClient(NewClientConfig("https://"+listener.Addr().String(), "app", testDeviceApiKey, testServerApiKey, time.Second))
	require.NoError(t, err)

	err = client.UnregisterDevice(context.Background(), "device")
	require.ErrorIs(t, err, ErrUnregisterDevice)
	require.ErrorContains(t, err, "remote error")
	require.False(t, isTemporary(err), err.Error())
}
