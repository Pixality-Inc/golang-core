package pusher

type RegisterTokenOptions struct {
	language *string
	locale   *string
	timezone *string
}

func NewRegisterTokenOptions() *RegisterTokenOptions {
	return &RegisterTokenOptions{}
}

type RegisterTokenOption func(options *RegisterTokenOptions)

func WithLanguage(language *string) RegisterTokenOption {
	return func(options *RegisterTokenOptions) {
		options.language = language
	}
}

func WithLocale(locale *string) RegisterTokenOption {
	return func(options *RegisterTokenOptions) {
		options.locale = locale
	}
}

func WithTimezone(timezone *string) RegisterTokenOption {
	return func(options *RegisterTokenOptions) {
		options.timezone = timezone
	}
}
