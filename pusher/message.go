package pusher

//nolint:modernize
type Message interface{}

type MessageImpl struct{}

func NewMessage() Message {
	return &MessageImpl{}
}
