package mailer

type Attachment interface {
	Name() string
}

type AttachmentImpl struct {
	name string
}

func NewAttachment(name string) *AttachmentImpl {
	return &AttachmentImpl{
		name: name,
	}
}

func (a *AttachmentImpl) Name() string {
	return a.name
}
