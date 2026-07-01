package pushwoosh

type RegisterDeviceOption = func(options *RegisterDeviceOptions)

type RegisterDeviceOptions struct {
	language *string
	timezone *int
}

func NewRegisterDeviceOptions() *RegisterDeviceOptions {
	return &RegisterDeviceOptions{}
}

func WithLanguage(language string) RegisterDeviceOption {
	return func(options *RegisterDeviceOptions) {
		options.language = &language
	}
}

func WithTimezone(timezone int) RegisterDeviceOption {
	return func(options *RegisterDeviceOptions) {
		options.timezone = &timezone
	}
}
