package pushwoosh

type ContentPriorityType string

const (
	ContentPriorityTypeUnspecified ContentPriorityType = "PRIORITY_UNSPECIFIED"
	ContentPriorityTypeMin         ContentPriorityType = "PRIORITY_MIN"
	ContentPriorityTypeLow         ContentPriorityType = "PRIORITY_LOW"
	ContentPriorityTypeDefault     ContentPriorityType = "PRIORITY_DEFAULT"
	ContentPriorityTypeHigh        ContentPriorityType = "PRIORITY_HIGH"
	ContentPriorityTypeMax         ContentPriorityType = "PRIORITY_MAX"
)

type ContentDeliveryPriorityType string

const (
	ContentDeliveryPriorityTypeNormal ContentDeliveryPriorityType = "NORMAL"
	ContentDeliveryPriorityTypeHigh   ContentDeliveryPriorityType = "HIGH"
)

type (
	LocalizedContent struct {
		Title            *string                      `json:"title,omitempty"`
		Subtitle         *string                      `json:"subtitle,omitempty"`
		Body             *string                      `json:"body,omitempty"`
		Icon             *string                      `json:"icon,omitempty"`
		Image            *string                      `json:"image,omitempty"`
		Attachment       *string                      `json:"attachment,omitempty"`
		Sound            *string                      `json:"sound,omitempty"`
		SoundEnabled     bool                         `json:"sound_enabled,omitempty"`
		Badges           *string                      `json:"badges,omitempty"`
		Banner           *string                      `json:"banner,omitempty"`
		Priority         *ContentPriorityType         `json:"priority,omitempty"`
		DeliveryPriority *ContentDeliveryPriorityType `json:"delivery_priority,omitempty"`
	}

	MessagePayloadContent struct {
		LocalizedContent map[string]map[ContentPlatformType]LocalizedContent `json:"localized_content"`
	}

	OpenActionRichMedia struct {
		Code *string `json:"code,omitempty"`
		Url  *string `json:"url,omitempty"`
	}

	OpenActionDeepLink struct {
		Code   string         `json:"code"`
		Params map[string]any `json:"params,omitempty"`
	}

	OpenActionLink struct {
		Url string `json:"url"`
	}

	OpenAction struct {
		RichMedia *OpenActionRichMedia `json:"rich_media,omitempty"`
		DeepLink  *OpenActionDeepLink  `json:"deep_link,omitempty"`
		Link      *OpenActionLink      `json:"link,omitempty"`
	}

	MessagePayload struct {
		Preset      *PresetId                          `json:"preset,omitempty"`
		Content     MessagePayloadContent              `json:"content"`
		Silent      bool                               `json:"silent,omitempty"`
		CustomData  map[string]any                     `json:"custom_data,omitempty"`
		OpenAction  *OpenAction                        `json:"open_action,omitempty"`
		OpenActions map[ContentPlatformType]OpenAction `json:"open_actions,omitempty"`
	}
)
