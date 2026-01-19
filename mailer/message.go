package mailer

type Message struct {
	From        Account
	To          []Account
	Cc          []Account
	Bcc         []Account
	Subject     string
	Body        Body
	Attachments []Attachment
}

func NewMessage() *Message {
	return &Message{
		From:        nil,
		To:          make([]Account, 0),
		Cc:          make([]Account, 0),
		Bcc:         make([]Account, 0),
		Subject:     "",
		Body:        nil,
		Attachments: make([]Attachment, 0),
	}
}

func (m *Message) WithFrom(account Account) *Message {
	m.From = account

	return m
}

func (m *Message) WithTo(accounts ...Account) *Message {
	m.To = append(m.To, accounts...)

	return m
}

func (m *Message) WithCc(accounts ...Account) *Message {
	m.Cc = append(m.Cc, accounts...)

	return m
}

func (m *Message) WithBcc(accounts ...Account) *Message {
	m.Bcc = append(m.Bcc, accounts...)

	return m
}

func (m *Message) WithSubject(subject string) *Message {
	m.Subject = subject

	return m
}

func (m *Message) WithBody(body Body) *Message {
	m.Body = body

	return m
}

func (m *Message) WithAttachments(attachments ...Attachment) *Message {
	m.Attachments = append(m.Attachments, attachments...)

	return m
}
