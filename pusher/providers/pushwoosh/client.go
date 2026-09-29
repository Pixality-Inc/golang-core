package pushwoosh

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"syscall"
	"time"

	http "github.com/pixality-inc/golang-core/http_client"
	"github.com/pixality-inc/golang-core/logger"
	"github.com/pixality-inc/golang-core/pusher"
	"github.com/valyala/fasthttp"
)

const (
	defaultTimeout = 30 * time.Second

	authorizationHeader = "Authorization"
	authorizationScheme = "Token "

	// the op crypto/tls sets on the *net.OpError of an alert received from the server
	tlsRemoteErrorOp = "remote error"
)

var (
	ErrRegisterDevice     = errors.New("register device")
	ErrUnregisterDevice   = errors.New("unregister device")
	ErrNotify             = errors.New("notify")
	ErrUnknownMessageType = errors.New("unknown message type")
	ErrScheduleRequired   = errors.New("schedule required")
	ErrEmptyResponse      = errors.New("empty response")

	errRegisterDeviceFailed   = errors.New("failed to register device")
	errUnregisterDeviceFailed = errors.New("failed to unregister device")
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

	timeout := config.Timeout()
	if timeout <= 0 {
		timeout = defaultTimeout
	}

	// no base authorization header: the token depends on the api method, see ClientConfig.
	// the write timeout bounds the tls handshake: without it fasthttp runs the handshake lazily
	// inside the request write and waits for the server hello with no deadline at all
	httpConfig := http.ConfigYaml{
		BaseUrlValue:      config.BaseApiUrl(),
		NameValue:         "pushwoosh",
		TimeoutValue:      timeout,
		WriteTimeoutValue: &timeout,
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
		c.deviceAuthorization(),
	)
	if err != nil {
		return wrapRequestError(ErrRegisterDevice, httpResponse, err)
	}

	return checkDeviceApiResponse[RegisterDeviceResponse](ErrRegisterDevice, errRegisterDeviceFailed, httpResponse)
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
		c.deviceAuthorization(),
	)
	if err != nil {
		return wrapRequestError(ErrUnregisterDevice, httpResponse, err)
	}

	return checkDeviceApiResponse[UnregisterDeviceResponse](ErrUnregisterDevice, errUnregisterDeviceFailed, httpResponse)
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
			schedule.After = new(fmt.Sprintf("%fs", requestOptions.SendAfter.Seconds()))
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
			return nil, wrapRequestError(ErrNotify, httpResponse, err)
		}

		var response *ApiResult[NotifyResponse]

		if err = httpResponse.DecodeJSON(&response); err != nil {
			return nil, fmt.Errorf("%w: %w", ErrNotify, err)
		}

		if response == nil {
			return nil, fmt.Errorf("%w: %w", ErrNotify, ErrEmptyResponse)
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
			c.serverAuthorization(),
		)

		return checkResponse(httpResponse, err)

	case MessageTypeMarketing:
		httpResponse, err := c.httpClient.Post(
			ctx,
			"/messaging/v2/notify",
			http.WithJsonBody(NotifySegmentRequest{
				Segment: notify,
			}),
			c.serverAuthorization(),
		)

		return checkResponse(httpResponse, err)

	default:
		return nil, fmt.Errorf("%w: %s", ErrUnknownMessageType, messageType)
	}
}

func (c *ClientImpl) deviceAuthorization() http.RequestOption {
	return http.WithHeader(authorizationHeader, authorizationScheme+c.config.DeviceApiKey())
}

func (c *ClientImpl) serverAuthorization() http.RequestOption {
	return http.WithHeader(authorizationHeader, authorizationScheme+c.config.ServerApiKey())
}

// checkDeviceApiResponse reads the device api envelope: pushwoosh reports its own failures with
// http 200 and an error status_code in the body
func checkDeviceApiResponse[T any](baseErr error, failureErr error, httpResponse http.Response) error {
	var response *ApiResponse[T]

	if err := httpResponse.DecodeJSON(&response); err != nil {
		return fmt.Errorf("%w: %w", baseErr, err)
	}

	if response == nil {
		return fmt.Errorf("%w: %w", baseErr, ErrEmptyResponse)
	}

	if response.StatusCode == fasthttp.StatusOK {
		return nil
	}

	apiErr := fmt.Errorf("%w: %s", failureErr, response.StatusMessage)

	if isTemporaryStatusCode(response.StatusCode) {
		return fmt.Errorf("%w: %w: %w", baseErr, pusher.ErrTemporary, apiErr)
	}

	return fmt.Errorf("%w: %w", baseErr, apiErr)
}

// wrapRequestError marks the failures of the request itself that may pass on a later attempt
func wrapRequestError(baseErr error, httpResponse http.Response, err error) error {
	if isTemporaryRequestError(httpResponse, err) {
		return fmt.Errorf("%w: %w: %w", baseErr, pusher.ErrTemporary, err)
	}

	return fmt.Errorf("%w: %w", baseErr, err)
}

// isTemporaryRequestError trusts the response status only for the errors the http client derived
// from it: on a transport error the response carries the default status 200 of an unfilled response.
// of the rest only network failures count: a malformed url or an unsupported scheme, a tls failure
// (verified locally or reported by the server) or a canceled context stay the same on every attempt.
// an unresolvable host does count: for a fixed provider host it is far more often a dns blip than a
// typo, and the caller bounds the repeats anyway
func isTemporaryRequestError(httpResponse http.Response, err error) bool {
	isStatusError := errors.Is(err, http.ErrNon200HttpCode) ||
		errors.Is(err, http.ErrNotFound) ||
		errors.Is(err, http.ErrBadRequest)

	if isStatusError && httpResponse != nil {
		return isTemporaryStatusCode(httpResponse.GetStatusCode())
	}

	return isNetworkError(err)
}

func isNetworkError(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// crypto/tls wraps a tls alert of the server (protocol version, handshake failure) into a
	// *net.OpError, which would pass for a network failure below
	var opErr *net.OpError
	if errors.As(err, &opErr) && opErr.Op == tlsRemoteErrorOp {
		return false
	}

	// dial errors, dns errors and *net.OpError (broken pipe, reset); fasthttp.ErrTimeout is not one
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}

	return errors.Is(err, fasthttp.ErrTimeout) ||
		errors.Is(err, fasthttp.ErrConnectionClosed) ||
		errors.Is(err, fasthttp.ErrDialTimeout) ||
		errors.Is(err, fasthttp.ErrTLSHandshakeTimeout) ||
		errors.Is(err, fasthttp.ErrNoFreeConns) ||
		errors.Is(err, io.EOF) ||
		errors.Is(err, io.ErrUnexpectedEOF) ||
		errors.Is(err, syscall.ECONNRESET) ||
		errors.Is(err, syscall.ECONNREFUSED) ||
		errors.Is(err, syscall.EPIPE)
}

func isTemporaryStatusCode(statusCode int) bool {
	return statusCode >= fasthttp.StatusInternalServerError ||
		statusCode == fasthttp.StatusTooManyRequests ||
		statusCode == fasthttp.StatusRequestTimeout
}
