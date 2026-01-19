package smtp

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/pixality-inc/golang-core/logger"
	"github.com/pixality-inc/golang-core/mailer"
	"github.com/wneessen/go-mail"
)

var (
	errClient               = errors.New("client")
	errSend                 = errors.New("send")
	errGetContent           = errors.New("get content")
	errTemplateNotSupported = errors.New("template not supported")
)

type Smtp struct {
	log    logger.Loggable
	config ConfigYaml
	client *mail.Client
}

func New(config ConfigYaml) (*Smtp, error) {
	options := make([]mail.Option, 0)

	options = append(options, mail.WithPort(config.Port()))

	username := config.Username()
	password := config.Password()

	if username != "" && password != "" {
		options = append(
			options,
			mail.WithSMTPAuth(mail.SMTPAuthPlain),
			mail.WithUsername(username),
			mail.WithPassword(password),
		)
	} else {
		options = append(options, mail.WithSMTPAuth(mail.SMTPAuthNoAuth))
	}

	client, err := mail.NewClient(
		config.Host(),
		options...,
	)
	if err != nil {
		return nil, errors.Join(errClient, err)
	}

	smtp := &Smtp{
		log:    logger.NewLoggableImplWithService("smtp"),
		config: config,
		client: client,
	}

	return smtp, nil
}

func (s *Smtp) Send(
	ctx context.Context,
	message *mailer.Message,
) (mailer.Result, error) {
	msg := mail.NewMsg()

	if err := msg.From(message.From.String()); err != nil {
		return nil, err
	}

	toEmails := make([]string, 0, len(message.To))

	for _, toAccount := range message.To {
		toEmails = append(toEmails, toAccount.String())
	}

	if err := msg.To(toEmails...); err != nil {
		return nil, err
	}

	ccEmails := make([]string, 0, len(message.Cc))

	for _, ccAccount := range message.Cc {
		ccEmails = append(ccEmails, ccAccount.String())
	}

	if err := msg.Cc(ccEmails...); err != nil {
		return nil, err
	}

	bccEmails := make([]string, 0, len(message.Bcc))

	for _, bccAccount := range message.Bcc {
		bccEmails = append(bccEmails, bccAccount.String())
	}

	if err := msg.Bcc(bccEmails...); err != nil {
		return nil, err
	}

	msg.Subject(message.Subject)

	switch body := message.Body.(type) {
	case *mailer.TemplateBody:
		return nil, errTemplateNotSupported

	default:
		content, err := body.Content(ctx)
		if err != nil {
			return nil, errors.Join(errGetContent, err)
		}

		var contentType mail.ContentType

		switch body.Type() {
		case mailer.BodyTypeText:
			contentType = mail.TypeTextPlain
		default:
			contentType = mail.TypeTextHTML
		}

		msg.SetBodyString(contentType, content)
	}

	if err := s.client.DialAndSend(msg); err != nil {
		return nil, errors.Join(errSend, err)
	}

	result := mailer.NewResult(uuid.NewString())

	return result, nil
}
