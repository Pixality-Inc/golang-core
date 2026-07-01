package pushwoosh

import (
	"context"
	"errors"
	"fmt"

	http "github.com/pixality-inc/golang-core/http_client"
	"github.com/pixality-inc/golang-core/logger"
)

var (
	ErrRegisterDevice   = errors.New("register device")
	ErrUnregisterDevice = errors.New("unregister device")
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
		options ...NotifyOption,
	)
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

	if response.StatusCode != 200 {
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

	if response.StatusCode != 200 {
		return fmt.Errorf("%w: failed to unregister device: %s", ErrUnregisterDevice, response.StatusMessage)
	}

	return nil
}
