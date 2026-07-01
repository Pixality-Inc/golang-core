package pushwoosh

type NotifyOption = func(options *NotifyOptions)

type NotifyOptions struct{}

func NewNotifyOptions() *NotifyOptions {
	return &NotifyOptions{}
}
