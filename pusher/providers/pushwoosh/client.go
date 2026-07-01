package pushwoosh

import (
	"context"
	"errors"
	"fmt"
	"time"

	http "github.com/pixality-inc/golang-core/http_client"
	"github.com/pixality-inc/golang-core/logger"
	"github.com/valyala/fasthttp"
)

var (
	ErrRegisterDevice     = errors.New("register device")
	ErrUnregisterDevice   = errors.New("unregister device")
	ErrNotify             = errors.New("notify")
	ErrUnknownMessageType = errors.New("unknown message type")
	ErrScheduleRequired   = errors.New("schedule required")
)

type Client interface {
	RegisterDevice(
		ctx context.Context,
		deviceType DeviceType,
		userId string,
		deviceId string,
		token string,
		options ...RegisterDeviceOption,
	) error

	UnregisterDevice(
		ctx context.Context,
		deviceId string,
	) error

	Notify(
		ctx context.Context,
		platforms []PlatformType,
		messageType MessageType,
		payload MessagePayload,
		options ...NotifyOption,
	) (*NotifyResult, error)
}

type ClientImpl struct {
	log        logger.Loggable
	config     ClientConfig
	httpClient http.Client
}

func NewClient(config ClientConfig) (Client, error) {
	log := logger.NewLoggableImplWithService("pushwoosh")

	httpConfig := http.ConfigYaml{
		BaseUrlValue: config.BaseApiUrl(),
		NameValue:    "pushwoosh",
		BaseHeadersValue: map[string][]string{
			"Authorization": {"Token " + config.ApiKey()},
		},
	}

	httpClient, err := http.NewClientImpl(log, &httpConfig)
	if err != nil {
		return nil, err
	}

	clientImpl := &ClientImpl{
		log:        log,
		config:     config,
		httpClient: httpClient,
	}

	return clientImpl, nil
}

func (c *ClientImpl) RegisterDevice(
	ctx context.Context,
	deviceType DeviceType,
	userId string,
	deviceId string,
	token string,
	options ...RegisterDeviceOption,
) error {
	requestOptions := NewRegisterDeviceOptions()

	for _, option := range options {
		option(requestOptions)
	}

	apiRequest := ApiRequest[RegisterDeviceRequest]{
		Request: RegisterDeviceRequest{
			Application: c.config.ApplicationId(),
			PushToken:   new(token),
			HwId:        deviceId,
			Timezone:    requestOptions.timezone,
			DeviceType:  int(deviceType),
			Language:    requestOptions.language,
			UserId:      new(userId),
		},
	}

	httpResponse, err := c.httpClient.Post(
		ctx,
		"/json/1.3/registerDevice",
		http.WithJsonBody(apiRequest),
	)
	if err != nil {
		return errors.Join(ErrRegisterDevice, err)
	}

	var response *ApiResponse[RegisterDeviceResponse]

	if err = httpResponse.DecodeJSON(&response); err != nil {
		return errors.Join(ErrRegisterDevice, err)
	}

	if response.StatusCode != fasthttp.StatusOK {
		return fmt.Errorf("%w: failed to register device: %s", ErrRegisterDevice, response.StatusMessage)
	}

	return nil
}

func (c *ClientImpl) UnregisterDevice(
	ctx context.Context,
	deviceId string,
) error {
	apiRequest := ApiRequest[UnregisterDeviceRequest]{
		Request: UnregisterDeviceRequest{
			Application: c.config.ApplicationId(),
			HwId:        deviceId,
		},
	}

	httpResponse, err := c.httpClient.Post(
		ctx,
		"/json/1.3/unregisterDevice",
		http.WithJsonBody(apiRequest),
	)
	if err != nil {
		return errors.Join(ErrUnregisterDevice, err)
	}

	var response *ApiResponse[UnregisterDeviceResponse]

	if err = httpResponse.DecodeJSON(&response); err != nil {
		return errors.Join(ErrUnregisterDevice, err)
	}

	if response.StatusCode != fasthttp.StatusOK {
		return fmt.Errorf("%w: failed to unregister device: %s", ErrUnregisterDevice, response.StatusMessage)
	}

	return nil
}

func (c *ClientImpl) Notify(
	ctx context.Context,
	platforms []PlatformType,
	messageType MessageType,
	payload MessagePayload,
	options ...NotifyOption,
) (*NotifyResult, error) {
	requestOptions := NewNotifyOptions()

	for _, option := range options {
		option(requestOptions)
	}

	var (
		usersList      *List
		hwIdsList      *List
		pushTokensList *List
	)

	if requestOptions.UsersIds != nil {
		usersList = NewList(requestOptions.UsersIds...)
	}

	if requestOptions.DevicesIds != nil {
		hwIdsList = NewList(requestOptions.DevicesIds...)
	}

	if requestOptions.PushTokens != nil {
		pushTokensList = NewList(requestOptions.PushTokens...)
	}

	var schedule *Schedule

	if requestOptions.SendAt != nil || requestOptions.SendAfter != nil {
		schedule = &Schedule{}

		if requestOptions.SendAt != nil {
			schedule.At = new(requestOptions.SendAt.In(time.UTC).Format(time.RFC3339))
		}

		if requestOptions.SendAfter != nil {
			schedule.At = new(fmt.Sprintf("%fs", requestOptions.SendAfter.Seconds()))
		}
	} else {
		return nil, ErrScheduleRequired
	}

	notify := Notify{
		Application: c.config.ApplicationId(),
		Platforms:   platforms,
		Users:       usersList,
		HwIds:       hwIdsList,
		PushTokens:  pushTokensList,
		Payload:     payload,
		MessageType: messageType,
		Schedule:    schedule,
	}

	checkResponse := func(httpResponse http.Response, err error) (*NotifyResult, error) {
		if err != nil {
			return nil, errors.Join(ErrNotify, err)
		}

		var response *ApiResult[NotifyResponse]

		if err = httpResponse.DecodeJSON(&response); err != nil {
			return nil, errors.Join(ErrNotify, err)
		}

		notifyResult := &NotifyResult{
			MessageId: response.Result.MessageCode,
		}

		return notifyResult, nil
	}

	switch messageType {
	case MessageTypeTransactional:
		httpResponse, err := c.httpClient.Post(
			ctx,
			"/messaging/v2/notify",
			http.WithJsonBody(NotifyTransactionalRequest{
				Transactional: notify,
			}),
		)

		return checkResponse(httpResponse, err)

	case MessageTypeMarketing:
		httpResponse, err := c.httpClient.Post(
			ctx,
			"/messaging/v2/notify",
			http.WithJsonBody(NotifySegmentRequest{
				Segment: notify,
			}),
		)

		return checkResponse(httpResponse, err)

	default:
		return nil, fmt.Errorf("%w: %s", ErrUnknownMessageType, messageType)
	}
}
