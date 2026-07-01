package pusher

import (
	"context"

	"github.com/pixality-inc/golang-core/logger"
)

//nolint:iface
type Pusher interface {
	RegisterToken(ctx context.Context, userId UserId, deviceId DeviceId, token Token, options ...RegisterTokenOption) error
	UnregisterToken(ctx context.Context, userId UserId, deviceId DeviceId, token Token) error
	SendMessageByUserId(ctx context.Context, userId *UserId, message Message) error
	SendMessageByDeviceId(ctx context.Context, deviceId *DeviceId, message Message) error
}

type Impl struct {
	Provider Provider

	log logger.Loggable
}

func New(provider Provider) *Impl {
	return &Impl{
		Provider: provider,
		log:      logger.NewLoggableImplWithService("pusher"),
	}
}
