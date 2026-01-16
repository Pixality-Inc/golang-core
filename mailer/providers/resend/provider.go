package resend

import (
	"context"
	"errors"

	"github.com/pixality-inc/golang-core/logger"
	"github.com/pixality-inc/golang-core/mailer"
	resendGo "github.com/resend/resend-go/v3"
)

var (
	errGetContent = errors.New("get content")
	errSend       = errors.New("send")
)

type Resend struct {
	log    logger.Loggable
	config ConfigYaml
	client *resendGo.Client
}

func New(config ConfigYaml) *Resend {
	return &Resend{
		log:    logger.NewLoggableImplWithService("resend"),
		config: config,
		client: resendGo.NewClient(config.ApiKey()),
	}
}

func (r *Resend) Send(
	ctx context.Context,
	message *mailer.Message,
) (mailer.Result, error) {
	toEmails := make([]string, 0, len(message.To))

	for _, toAccount := range message.To {
		toEmails = append(toEmails, toAccount.String())
	}

	params := &resendGo.SendEmailRequest{
		From:    message.From.String(),
		To:      toEmails,
		Subject: message.Subject,
	}

	for _, ccAccount := range message.Cc {
		params.Cc = append(params.Cc, ccAccount.String())
	}

	for _, bccAccount := range message.Bcc {
		params.Bcc = append(params.Bcc, bccAccount.String())
	}

	switch body := message.Body.(type) {
	case *mailer.TemplateBody:
		params.Template = &resendGo.EmailTemplate{
			Id:        body.Name(),
			Variables: body.Variables(),
		}

	default:
		content, err := body.Content(ctx)
		if err != nil {
			return nil, errors.Join(errGetContent, err)
		}

		params.Html = content
	}

	email, err := r.client.Emails.Send(params)
	if err != nil {
		return nil, errors.Join(errSend, err)
	}

	result := mailer.NewResult(email.Id)

	return result, nil
}
