package websocket

import (
	"time"

	fasthttpWebsocket "github.com/fasthttp/websocket"
	"github.com/valyala/fasthttp"
)

type Option func(*options)

// message types of WithWriteMessageType
const (
	TextMessage   = fasthttpWebsocket.TextMessage
	BinaryMessage = fasthttpWebsocket.BinaryMessage
)

type options struct {
	upgrader         fasthttpWebsocket.FastHTTPUpgrader
	writeMessageType int
	readLimit        int64
	pingInterval     time.Duration
	pongTimeout      time.Duration
}

func WithCheckOrigin(checkOrigin func(ctx *fasthttp.RequestCtx) bool) Option {
	return func(o *options) {
		o.upgrader.CheckOrigin = checkOrigin
	}
}

func WithSubprotocols(subprotocols ...string) Option {
	return func(o *options) {
		o.upgrader.Subprotocols = subprotocols
	}
}

func WithCompression(enabled bool) Option {
	return func(o *options) {
		o.upgrader.EnableCompression = enabled
	}
}

func WithHandshakeTimeout(timeout time.Duration) Option {
	return func(o *options) {
		o.upgrader.HandshakeTimeout = timeout
	}
}

func WithReadBufferSize(size int) Option {
	return func(o *options) {
		o.upgrader.ReadBufferSize = size
	}
}

func WithWriteBufferSize(size int) Option {
	return func(o *options) {
		o.upgrader.WriteBufferSize = size
	}
}

func WithWriteMessageType(messageType int) Option {
	return func(o *options) {
		o.writeMessageType = messageType
	}
}

// WithReadLimit limits the size of an incoming message in bytes: a larger one closes the connection
// with CloseMessageTooBig
func WithReadLimit(limit int64) Option {
	return func(o *options) {
		o.readLimit = limit
	}
}

// WithKeepAlive pings the peer every pingInterval and closes the connection when nothing (a pong or
// a message) comes from the peer within pongTimeout. pongTimeout is longer than pingInterval
func WithKeepAlive(pingInterval time.Duration, pongTimeout time.Duration) Option {
	return func(o *options) {
		o.pingInterval = pingInterval
		o.pongTimeout = pongTimeout
	}
}

func defaultOptions() *options {
	return &options{
		upgrader:         fasthttpWebsocket.FastHTTPUpgrader{},
		writeMessageType: fasthttpWebsocket.BinaryMessage,
	}
}
