package pusher

import (
	"context"

	"github.com/pixality-inc/golang-core/logger"
)

//nolint:iface
type Pusher interface {
	RegisterToken(ctx context.Context, userId UserId, deviceId DeviceId, tokenType TokenType, token Token, options ...RegisterTokenOption) error
	UnregisterToken(ctx context.Context, userId UserId, deviceId DeviceId, token Token) error
	SendMessageByUserId(ctx context.Context, userId UserId, message Message) error
	SendMessageByDeviceId(ctx context.Context, deviceId DeviceId, message Message) error
}

type Impl struct {
	log      logger.Loggable
	provider Provider
}

func New(provider Provider) Pusher {
	return &Impl{
		log:      logger.NewLoggableImplWithService("pusher"),
		provider: provider,
	}
}

func (p *Impl) RegisterToken(ctx context.Context, userId UserId, deviceId DeviceId, tokenType TokenType, token Token, options ...RegisterTokenOption) error {
	return p.provider.RegisterToken(ctx, userId, deviceId, tokenType, token, options...)
}

func (p *Impl) UnregisterToken(ctx context.Context, userId UserId, deviceId DeviceId, token Token) error {
	return p.provider.UnregisterToken(ctx, userId, deviceId, token)
}

func (p *Impl) SendMessageByUserId(ctx context.Context, userId UserId, message Message) error {
	return p.provider.SendMessageByUserId(ctx, userId, message)
}

func (p *Impl) SendMessageByDeviceId(ctx context.Context, deviceId DeviceId, message Message) error {
	return p.provider.SendMessageByDeviceId(ctx, deviceId, message)
}
