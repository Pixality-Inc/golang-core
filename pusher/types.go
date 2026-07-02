package pusher

type TokenType string

const (
	TokenTypeIOS     TokenType = "ios"
	TokenTypeAndroid TokenType = "android"
	TokenTypeHuawei  TokenType = "huawei"
	TokenTypeChrome  TokenType = "chrome"
	TokenTypeSafari  TokenType = "safari"
	TokenTypeFirefox TokenType = "firefox"
)

type UserId string

type DeviceId string

type MessageId string

type SendMessageResult interface {
	MessageId() MessageId
}

type SendMessageResultImpl struct {
	messageId MessageId
}

func NewSendMessageResult(messageId MessageId) *SendMessageResultImpl {
	return &SendMessageResultImpl{
		messageId: messageId,
	}
}

func (r *SendMessageResultImpl) MessageId() MessageId {
	return r.messageId
}
