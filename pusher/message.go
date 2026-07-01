package pusher

type Message interface{}

type MessageImpl struct{}

func NewMessage() Message {
	return &MessageImpl{}
}
