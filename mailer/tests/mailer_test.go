package tests

import (
	"testing"

	"github.com/google/uuid"
	"github.com/pixality-inc/golang-core/mailer"
	mockProvider "github.com/pixality-inc/golang-core/mailer/mocks"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestMailer(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	testCases := []struct {
		name           string
		message        *mailer.Message
		mailerProvider func() mailer.Provider
		wantResultId   string
		wantErr        error
	}{
		{
			name: "success",
			message: mailer.NewMessage().
				WithTo(mailer.NewAccount("test@test.te")).
				WithFrom(mailer.NewAccount("foo@bar.baz")).
				WithSubject("Test message").
				WithBody(mailer.NewTextBody("Hello, World!")),
			wantResultId: "",
			wantErr:      nil,
		},
		{
			name: "no_from",
			message: mailer.NewMessage().
				WithTo(mailer.NewAccount("test@test.te")).
				WithSubject("Test message").
				WithBody(mailer.NewTextBody("Hello, World!")),
			wantErr: mailer.ErrNoFrom,
		},
		{
			name: "no_to",
			message: mailer.NewMessage().
				WithFrom(mailer.NewAccount("foo@bar.baz")).
				WithSubject("Test message").
				WithBody(mailer.NewTextBody("Hello, World!")),
			wantErr: mailer.ErrNoTo,
		},
		{
			name: "no_subject",
			message: mailer.NewMessage().
				WithTo(mailer.NewAccount("test@test.te")).
				WithFrom(mailer.NewAccount("foo@bar.baz")).
				WithBody(mailer.NewTextBody("Hello, World!")),
			wantErr: mailer.ErrNoSubject,
		},
		{
			name: "no_body",
			message: mailer.NewMessage().
				WithSubject("Test message").
				WithFrom(mailer.NewAccount("foo@bar.baz")).
				WithTo(mailer.NewAccount("test@test.te")).
				WithFrom(mailer.NewAccount("foo@bar.baz")),
			wantErr: mailer.ErrNoBody,
		},
		{
			name: "attachments_not_implemented",
			message: mailer.NewMessage().
				WithSubject("Test message").
				WithTo(mailer.NewAccount("test@test.te")).
				WithFrom(mailer.NewAccount("foo@bar.baz")).
				WithBody(mailer.NewTextBody("Hello, World!")).
				WithAttachments(mailer.NewAttachment("test.txt")),
			wantErr: mailer.ErrAttachmentsNotImplemented,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			t.Cleanup(ctrl.Finish)

			wantResultId := testCase.wantResultId
			if wantResultId == "" {
				wantResultId = uuid.NewString()
			}

			var mailerProvider mailer.Provider

			if testCase.mailerProvider == nil && testCase.wantErr == nil {
				provider := mockProvider.NewMockProvider(ctrl)
				provider.EXPECT().Send(ctx, testCase.message).Return(mailer.NewResult(wantResultId), nil)

				mailerProvider = provider
			}

			mailerService := mailer.New(mailerProvider)

			result, err := mailerService.Send(ctx, testCase.message)

			if testCase.wantErr != nil {
				require.ErrorIs(t, err, testCase.wantErr)
			} else {
				require.NoError(t, err)
				require.Equal(t, wantResultId, result.Id())
			}
		})
	}
}
