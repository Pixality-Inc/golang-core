package pushwoosh

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/pixality-inc/golang-core/clock"
	"github.com/pixality-inc/golang-core/logger"
	"github.com/pixality-inc/golang-core/pusher"
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
) (pusher.SendMessageResult, error) {
	return p.sendMessage(
		ctx,
		clock.GetClock(ctx).Now(),
		message,
		WithUsersId(string(userId)),
	)
}

func (p *PusherProvider) SendMessageByDeviceId(
	ctx context.Context,
	deviceId pusher.DeviceId,
	message pusher.Message,
) (pusher.SendMessageResult, error) {
	return p.sendMessage(
		ctx,
		clock.GetClock(ctx).Now(),
		message,
		WithDeviceId(string(deviceId)),
	)
}

func (p *PusherProvider) sendMessage(
	ctx context.Context,
	now time.Time,
	message pusher.Message,
	options ...NotifyOption,
) (pusher.SendMessageResult, error) {
	payload, err := messageToPayload(message)
	if err != nil {
		return nil, err
	}

	options = append(options, WithSendAt(now))

	notifyResult, err := p.pushwooshClient.Notify(
		ctx,
		AllPlatformTypes,
		MessageTypeTransactional,
		*payload,
		options...,
	)
	if err != nil {
		return nil, err
	}

	return pusher.NewSendMessageResult(
		pusher.MessageId(notifyResult.MessageId),
	), nil
}

func messageToPayload(message pusher.Message) (*MessagePayload, error) {
	nilIfEmpty := func(s string) *string {
		if s == "" {
			return nil
		}

		return &s
	}

	var badges *string

	badgesCount := message.Badges()

	if badgesCount > 0 {
		badges = new(strconv.Itoa(badgesCount))
	}

	localizedContent := LocalizedContent{
		Title:    nilIfEmpty(message.Title()),
		Subtitle: nilIfEmpty(message.Subtitle()),
		Body:     nilIfEmpty(message.Body()),
		Badges:   badges,
	}

	isSilent := message.Silent()

	var localizedContentMap map[string]map[ContentPlatformType]LocalizedContent

	if !isSilent {
		localizedContentMap = map[string]map[ContentPlatformType]LocalizedContent{
			"default": {
				ContentPlatformTypeIOS:     localizedContent,
				ContentPlatformTypeAndroid: localizedContent,
				ContentPlatformTypeHuawei:  localizedContent,
				ContentPlatformTypeChrome:  localizedContent,
				ContentPlatformTypeSafari:  localizedContent,
				ContentPlatformTypeFirefox: localizedContent,
				ContentPlatformTypeIE:      localizedContent,
			},
		}
	}

	payload := MessagePayload{
		Content: MessagePayloadContent{
			LocalizedContent: localizedContentMap,
		},
		Silent:     isSilent,
		CustomData: message.CustomData(),
	}

	return &payload, nil
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
