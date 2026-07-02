package pusher

type RegisterTokenOptions struct {
	Language *string
	Locale   *string
	Timezone *string
}

func NewRegisterTokenOptions() *RegisterTokenOptions {
	return &RegisterTokenOptions{}
}

type RegisterTokenOption func(options *RegisterTokenOptions)

func WithLanguage(language *string) RegisterTokenOption {
	return func(options *RegisterTokenOptions) {
		options.Language = language
	}
}

func WithLocale(locale *string) RegisterTokenOption {
	return func(options *RegisterTokenOptions) {
		options.Locale = locale
	}
}

func WithTimezone(timezone *string) RegisterTokenOption {
	return func(options *RegisterTokenOptions) {
		options.Timezone = timezone
	}
}
