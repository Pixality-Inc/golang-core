package mailer

import "context"

//go:generate mockgen -destination mocks/provider_gen.go -source provider.go
//nolint:iface
type Provider interface {
	Send(ctx context.Context, message *Message) (Result, error)
}
