package mailer

import (
	"context"
	"errors"

	"github.com/pixality-inc/golang-core/logger"
)

var (
	ErrNoFrom                    = errors.New("no from")
	ErrNoTo                      = errors.New("no to")
	ErrNoSubject                 = errors.New("no subject")
	ErrNoBody                    = errors.New("no body")
	ErrSend                      = errors.New("send")
	ErrAttachmentsNotImplemented = errors.New("attachments are not implemented yet")
)

//nolint:iface
type Mailer interface {
	Send(ctx context.Context, message *Message) (Result, error)
}

type Impl struct {
	log      logger.Loggable
	provider Provider
}

func New(provider Provider) *Impl {
	return &Impl{
		log:      logger.NewLoggableImplWithService("mailer"),
		provider: provider,
	}
}

func (m *Impl) Send(ctx context.Context, message *Message) (Result, error) {
	log := m.log.GetLogger(ctx)

	if message.From == nil {
		return nil, ErrNoFrom
	}

	if len(message.To) == 0 {
		return nil, ErrNoTo
	}

	if message.Subject == "" {
		return nil, ErrNoSubject
	}

	if message.Body == nil {
		return nil, ErrNoBody
	}

	if len(message.Attachments) > 0 {
		return nil, ErrAttachmentsNotImplemented
	}

	log.Debugf("sending mail %q from %s to %v", message.Subject, message.From.String(), message.To)

	result, err := m.provider.Send(ctx, message)
	if err != nil {
		return nil, errors.Join(ErrSend, err)
	}

	return result, nil
}
