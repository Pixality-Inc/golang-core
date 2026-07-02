package pushwoosh

type MessageType string

const (
	MessageTypeMarketing     MessageType = "MESSAGE_TYPE_MARKETING"
	MessageTypeTransactional MessageType = "MESSAGE_TYPE_TRANSACTIONAL"
)
