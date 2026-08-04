package resend

import (
	"context"
	"errors"
	"net/http"

	"github.com/pixality-inc/golang-core/logger"
	"github.com/pixality-inc/golang-core/mailer"
	resendGo "github.com/resend/resend-go/v3"
)

const sendEmailPath = "emails"

var (
	errGetContent   = errors.New("get content")
	errBuildRequest = errors.New("build request")
	errSend         = errors.New("send")
)

// sendEmailRequest repeats the fields of resendGo.SendEmailRequest used by this provider,
// the only difference is omitempty on subject. resend serializes subject unconditionally and
// a subject sent together with a template overrides the subject stored in that template,
// so an empty subject has to be left out of the payload entirely
type sendEmailRequest struct {
	From     string                  `json:"from"`
	To       []string                `json:"to"`
	Subject  string                  `json:"subject,omitempty"`
	Cc       []string                `json:"cc,omitempty"`
	Bcc      []string                `json:"bcc,omitempty"`
	Html     string                  `json:"html,omitempty"`
	Template *resendGo.EmailTemplate `json:"template,omitempty"`
}

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

	params := &sendEmailRequest{
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

	request, err := r.client.NewRequest(ctx, http.MethodPost, sendEmailPath, params)
	if err != nil {
		return nil, errors.Join(errBuildRequest, err)
	}

	email := new(resendGo.SendEmailResponse)

	//nolint:bodyclose // the resend client closes the response body on its own
	if _, err = r.client.Perform(request, email); err != nil {
		return nil, errors.Join(errSend, err)
	}

	result := mailer.NewResult(email.Id)

	return result, nil
}
