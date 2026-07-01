package pushwoosh

type ApiRequest[T any] struct {
	Request T `json:"request"`
}

type ApiResponse[T any] struct {
	StatusCode    int    `json:"status_code"`
	StatusMessage string `json:"status_message"`
	Response      T      `json:"response"`
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

type RegisterDeviceResponse struct {
	IosCategories []any `json:"ios_categories,omitempty"`
}

type UnregisterDeviceRequest struct {
	Application string `json:"application"`
	HwId        string `json:"hwid"`
}

type UnregisterDeviceResponse struct{}
