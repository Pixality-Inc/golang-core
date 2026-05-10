package net

type Message interface {
	Data() []byte
}

type MessageImpl struct {
	data []byte
}

func NewMessage(data []byte) *MessageImpl {
	return &MessageImpl{
		data: data,
	}
}

func (m *MessageImpl) Data() []byte {
	return m.data
}
