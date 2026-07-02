package pushwoosh

type ApiRequest[T any] struct {
	Request T `json:"request"`
}

type ApiResponse[T any] struct {
	StatusCode    int    `json:"status_code"`
	StatusMessage string `json:"status_message"`
	Response      T      `json:"response"`
}

type ApiResult[T any] struct {
	Result T `json:"result"`
}

type RegisterDeviceRequest struct {
	Application string         `json:"application"`
	Email       *string        `json:"email,omitempty"`
	PushToken   *string        `json:"push_token,omitempty"`
	HwId        string         `json:"hwid"`
	Idfa        *string        `json:"idfa,omitempty"`
	Timezone    *int           `json:"timezone,omitempty"`
	DeviceType  int            `json:"device_type"`
	Language    *string        `json:"language,omitempty"`
	UserId      *string        `json:"userId,omitempty"`
	AppVersion  *string        `json:"app_version,omitempty"`
	DeviceModel *string        `json:"device_model,omitempty"`
	OsVersion   *string        `json:"os_version,omitempty"`
	PublicKey   *string        `json:"public_key,omitempty"`
	AuthToken   *string        `json:"auth_token,omitempty"`
	FcmToken    *string        `json:"fcm_token,omitempty"`
	FcmPushSet  *string        `json:"fcm_push_set,omitempty"`
	Tags        map[string]any `json:"tags,omitempty"`
}

type RegisterDeviceResponse struct{}

type UnregisterDeviceRequest struct {
	Application string `json:"application"`
	HwId        string `json:"hwid"`
}

type UnregisterDeviceResponse struct{}

type List struct {
	List []string `json:"list"`
}

func NewList(items ...string) *List {
	return &List{
		List: items,
	}
}

type Schedule struct {
	At    *string `json:"at,omitempty"`
	After *string `json:"after,omitempty"`
}

type Notify struct {
	Application              string         `json:"application"`
	Platforms                []PlatformType `json:"platforms,omitempty"`
	Users                    *List          `json:"users,omitempty"`
	HwIds                    *List          `json:"hwids,omitempty"`
	PushTokens               *List          `json:"push_tokens,omitempty"`
	Payload                  MessagePayload `json:"payload"`
	MessageType              MessageType    `json:"message_type"`
	ReturnUnknownIdentifiers bool           `json:"return_unknown_identifiers,omitempty"`
	UseLatestUserDevice      bool           `json:"use_latest_user_device,omitempty"`
	Schedule                 *Schedule      `json:"schedule,omitempty"`
}

type NotifyTransactionalRequest struct {
	Transactional Notify `json:"transactional"`
}

type NotifySegmentRequest struct {
	Segment Notify `json:"segment"`
}

type NotifyResponse struct {
	MessageCode        string   `json:"message_code"`
	UnknownIdentifiers []string `json:"unknown_identifiers"`
}
