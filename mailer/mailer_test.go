package mailer_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/pixality-inc/golang-core/mailer"
	mockProvider "github.com/pixality-inc/golang-core/mailer/mocks"
	"github.com/pixality-inc/golang-core/util"
)

var errProvider = errors.New("provider failed")

func TestMailerSend(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name    string
		message *mailer.Message
		wantErr error
	}{
		{
			name: "success",
			message: mailer.NewMessage().
				WithFrom(mailer.NewAccount("foo@bar.baz")).
				WithTo(mailer.NewAccount("test@test.te")).
				WithSubject("Test message").
				WithBody(mailer.NewTextBody("Hello, World!")),
			wantErr: nil,
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
				WithFrom(mailer.NewAccount("foo@bar.baz")).
				WithTo(mailer.NewAccount("test@test.te")).
				WithBody(mailer.NewTextBody("Hello, World!")),
			wantErr: mailer.ErrNoSubject,
		},
		{
			name: "no_body",
			message: mailer.NewMessage().
				WithFrom(mailer.NewAccount("foo@bar.baz")).
				WithTo(mailer.NewAccount("test@test.te")).
				WithSubject("Test message"),
			wantErr: mailer.ErrNoBody,
		},
		{
			name: "attachments_not_implemented",
			message: mailer.NewMessage().
				WithFrom(mailer.NewAccount("foo@bar.baz")).
				WithTo(mailer.NewAccount("test@test.te")).
				WithSubject("Test message").
				WithBody(mailer.NewTextBody("Hello, World!")).
				WithAttachments(mailer.NewAttachment("test.txt")),
			wantErr: mailer.ErrAttachmentsNotImplemented,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			var provider mailer.Provider

			if testCase.wantErr == nil {
				mock := mockProvider.NewMockProvider(gomock.NewController(t))
				mock.EXPECT().Send(ctx, testCase.message).Return(mailer.NewResult("result-id"), nil)

				provider = mock
			}

			result, err := mailer.New(provider).Send(ctx, testCase.message)

			if testCase.wantErr != nil {
				require.ErrorIs(t, err, testCase.wantErr)

				return
			}

			require.NoError(t, err)
			require.Equal(t, "result-id", result.Id())
		})
	}
}

func TestMailerSendProviderError(t *testing.T) {
	t.Parallel()

	ctx := t.Context()

	message := mailer.NewMessage().
		WithFrom(mailer.NewAccount("from@example.com")).
		WithTo(mailer.NewAccount("to@example.com")).
		WithSubject("subject").
		WithBody(mailer.NewTextBody("body"))

	provider := mockProvider.NewMockProvider(gomock.NewController(t))
	provider.EXPECT().Send(ctx, message).Return(nil, errProvider)

	_, err := mailer.New(provider).Send(ctx, message)
	require.ErrorIs(t, err, mailer.ErrSend)
	require.ErrorIs(t, err, errProvider)
}

func TestAccount(t *testing.T) {
	t.Parallel()

	account := mailer.NewAccount("user@example.com")

	assert.Equal(t, "user@example.com", account.Email())
	assert.Empty(t, account.Name())
	assert.Equal(t, "user@example.com", account.String())

	named := account.WithName("John Doe")

	assert.Same(t, account, named)
	assert.Equal(t, "John Doe", named.Name())
	assert.Equal(t, "John Doe <user@example.com>", named.String())
}

func TestHtmlBody(t *testing.T) {
	t.Parallel()

	body := mailer.NewHtmlBody("<b>hi</b>")

	assert.Equal(t, mailer.BodyTypeHtml, body.Type())

	content, err := body.Content(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "<b>hi</b>", content)
}

func TestTextBody(t *testing.T) {
	t.Parallel()

	body := mailer.NewTextBody("plain text")

	assert.Equal(t, mailer.BodyTypeText, body.Type())

	content, err := body.Content(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "plain text", content)
}

func TestTemplateBody(t *testing.T) {
	t.Parallel()

	variables := map[string]any{"key": "value"}
	body := mailer.NewTemplateBody("tpl-1", variables)

	assert.Equal(t, mailer.BodyTypeTemplate, body.Type())
	assert.Equal(t, "tpl-1", body.Name())
	assert.Equal(t, variables, body.Variables())

	_, err := body.Content(t.Context())
	require.ErrorIs(t, err, util.ErrNotImplemented)
}

func TestGoTemplateBody(t *testing.T) {
	t.Parallel()

	body := mailer.NewGoTemplateBody("greeting", "Hello, {{.Name}}!", map[string]any{"Name": "John"})

	assert.Equal(t, mailer.BodyTypeGoTemplate, body.Type())

	content, err := body.Content(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "Hello, John!", content)
}

func TestGoTemplateBodyNilVariables(t *testing.T) {
	t.Parallel()

	body := mailer.NewGoTemplateBody("static", "no variables", nil)

	content, err := body.Content(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "no variables", content)
}

func TestGoTemplateBodyParseError(t *testing.T) {
	t.Parallel()

	body := mailer.NewGoTemplateBody("broken", "Hello, {{.Name", nil)

	_, err := body.Content(t.Context())
	require.ErrorContains(t, err, "template parse")
}

func TestGoTemplateBodyExecuteError(t *testing.T) {
	t.Parallel()

	body := mailer.NewGoTemplateBody("broken", "{{.Name.Field}}", map[string]any{"Name": "John"})

	_, err := body.Content(t.Context())
	require.ErrorContains(t, err, "template execute")
}

func TestMessageBuilder(t *testing.T) {
	t.Parallel()

	from := mailer.NewAccount("from@example.com")
	firstTo := mailer.NewAccount("to1@example.com")
	secondTo := mailer.NewAccount("to2@example.com")
	ccAccount := mailer.NewAccount("cc@example.com")
	bccAccount := mailer.NewAccount("bcc@example.com")
	body := mailer.NewTextBody("body")
	attachment := mailer.NewAttachment("file.txt")

	message := mailer.NewMessage().
		WithFrom(from).
		WithTo(firstTo).
		WithTo(secondTo).
		WithCc(ccAccount).
		WithBcc(bccAccount).
		WithSubject("subject").
		WithBody(body).
		WithAttachments(attachment)

	assert.Equal(t, mailer.Account(from), message.From)
	assert.Equal(t, []mailer.Account{firstTo, secondTo}, message.To)
	assert.Equal(t, []mailer.Account{ccAccount}, message.Cc)
	assert.Equal(t, []mailer.Account{bccAccount}, message.Bcc)
	assert.Equal(t, "subject", message.Subject)
	assert.Equal(t, mailer.Body(body), message.Body)
	assert.Equal(t, []mailer.Attachment{attachment}, message.Attachments)
}

func TestResult(t *testing.T) {
	t.Parallel()

	result := mailer.NewResult("id-123")

	assert.Equal(t, "id-123", result.Id())
}

func TestAttachment(t *testing.T) {
	t.Parallel()

	attachment := mailer.NewAttachment("report.pdf")

	assert.Equal(t, "report.pdf", attachment.Name())
}
