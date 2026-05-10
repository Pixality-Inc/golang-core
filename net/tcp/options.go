package tcp

import "crypto/tls"

type Option func(*options)

type options struct {
	tlsConfig *tls.Config
}

func WithTLSConfig(config *tls.Config) Option {
	return func(o *options) {
		o.tlsConfig = config
	}
}
