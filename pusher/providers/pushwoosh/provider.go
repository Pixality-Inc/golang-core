package pushwoosh

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/pixality-inc/golang-core/logger"
	"github.com/pixality-inc/golang-core/pusher"
	"github.com/pixality-inc/golang-core/util"
)

var (
	ErrUnknownTokenType = errors.New("unknown token type")
	ErrUnknownTimezone  = errors.New("unknown timezone")
)

type PusherProvider struct {
	log             logger.Loggable
	pushwooshClient Client
}

func NewPusherProvider(pushwooshClient Client) *PusherProvider {
	return &PusherProvider{
		log:             logger.NewLoggableImplWithService("pushwoosh_provider"),
		pushwooshClient: pushwooshClient,
	}
}

func (p *PusherProvider) RegisterToken(
	ctx context.Context,
	userId pusher.UserId,
	deviceId pusher.DeviceId,
	tokenType pusher.TokenType,
	token pusher.Token,
	options ...pusher.RegisterTokenOption,
) error {
	deviceType, err := convertTokenType(tokenType)
	if err != nil {
		return err
	}

	registerTokenOptions := pusher.NewRegisterTokenOptions()

	for _, option := range options {
		option(registerTokenOptions)
	}

	registerDeviceOptions := make([]RegisterDeviceOption, 0)

	if registerTokenOptions.Language != nil {
		registerDeviceOptions = append(registerDeviceOptions, WithLanguage(*registerTokenOptions.Language))
	}

	if registerTokenOptions.Timezone != nil {
		zoneOffsetSeconds, err := getTimezoneOffsetSeconds(*registerTokenOptions.Timezone)
		if err != nil {
			return err
		}

		registerDeviceOptions = append(registerDeviceOptions, WithTimezone(zoneOffsetSeconds))
	}

	return p.pushwooshClient.RegisterDevice(
		ctx,
		deviceType,
		string(userId),
		string(deviceId),
		string(token),
		registerDeviceOptions...,
	)
}

func (p *PusherProvider) UnregisterToken(
	ctx context.Context,
	_ pusher.UserId,
	deviceId pusher.DeviceId,
	_ pusher.Token,
) error {
	return p.pushwooshClient.UnregisterDevice(
		ctx,
		string(deviceId),
	)
}

func (p *PusherProvider) SendMessageByUserId(
	ctx context.Context,
	userId pusher.UserId,
	message pusher.Message,
) error {
	// @todo!!!
	return util.ErrNotImplemented
}

func (p *PusherProvider) SendMessageByDeviceId(
	ctx context.Context,
	deviceId pusher.DeviceId,
	message pusher.Message,
) error {
	// @todo!!!
	return util.ErrNotImplemented
}

func convertTokenType(tokenType pusher.TokenType) (DeviceType, error) {
	switch tokenType {
	case pusher.TokenTypeIOS:
		return DeviceTypeIOS, nil
	case pusher.TokenTypeAndroid:
		return DeviceTypeAndroid, nil
	case pusher.TokenTypeHuawei:
		return DeviceTypeHuawei, nil
	case pusher.TokenTypeChrome:
		return DeviceTypeChrome, nil
	case pusher.TokenTypeSafari:
		return DeviceTypeSafari, nil
	case pusher.TokenTypeFirefox:
		return DeviceTypeFirefox, nil
	default:
		return 0, fmt.Errorf("%w: %s", ErrUnknownTokenType, tokenType)
	}
}

func getTimezoneOffsetSeconds(timezone string) (int, error) {
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return 0, errors.Join(ErrUnknownTimezone, err)
	}

	_, offsetSeconds := time.Now().In(loc).Zone()

	return offsetSeconds, nil
}
