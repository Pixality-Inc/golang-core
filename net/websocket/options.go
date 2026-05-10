package websocket

import (
	"time"

	fasthttpWebsocket "github.com/fasthttp/websocket"
	"github.com/valyala/fasthttp"
)

type Option func(*options)

type options struct {
	upgrader         fasthttpWebsocket.FastHTTPUpgrader
	writeMessageType int
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

func defaultOptions() *options {
	return &options{
		upgrader:         fasthttpWebsocket.FastHTTPUpgrader{},
		writeMessageType: fasthttpWebsocket.BinaryMessage,
	}
}
