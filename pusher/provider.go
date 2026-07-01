package pusher

import "context"

//nolint:iface
type Provider interface {
	RegisterToken(ctx context.Context, userId UserId, deviceId DeviceId, tokenType TokenType, token Token, options ...RegisterTokenOption) error
	UnregisterToken(ctx context.Context, userId UserId, deviceId DeviceId, token Token) error
	SendMessageByUserId(ctx context.Context, userId UserId, message Message) error
	SendMessageByDeviceId(ctx context.Context, deviceId DeviceId, message Message) error
}
