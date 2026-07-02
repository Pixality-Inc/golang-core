package pushwoosh

import "time"

type NotifyOption = func(options *NotifyOptions)

type NotifyOptions struct {
	UsersIds   []string
	DevicesIds []string
	PushTokens []string
	SendAt     *time.Time
	SendAfter  *time.Duration
}

func NewNotifyOptions() *NotifyOptions {
	return &NotifyOptions{}
}

func WithUsersId(usersId string) NotifyOption {
	return WithUsersIds(usersId)
}

func WithUsersIds(usersIds ...string) NotifyOption {
	return func(options *NotifyOptions) {
		options.UsersIds = append(options.UsersIds, usersIds...)
	}
}

func WithDeviceId(devicesId string) NotifyOption {
	return WithDevicesIds(devicesId)
}

func WithDevicesIds(devicesIds ...string) NotifyOption {
	return func(options *NotifyOptions) {
		options.DevicesIds = append(options.DevicesIds, devicesIds...)
	}
}

func WithPushToken(pushToken string) NotifyOption {
	return WithPushTokens(pushToken)
}

func WithPushTokens(pushTokens ...string) NotifyOption {
	return func(options *NotifyOptions) {
		options.PushTokens = append(options.PushTokens, pushTokens...)
	}
}

func WithSendAt(sendAt time.Time) NotifyOption {
	return func(options *NotifyOptions) {
		options.SendAt = &sendAt
	}
}

func WithSendAfter(sendAfter time.Duration) NotifyOption {
	return func(options *NotifyOptions) {
		options.SendAfter = &sendAfter
	}
}
