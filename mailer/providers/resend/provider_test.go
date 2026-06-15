package resend

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	resendGo "github.com/resend/resend-go/v3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pixality-inc/golang-core/mailer"
)

func newTestProvider(t *testing.T, handler http.HandlerFunc) *Resend {
	t.Helper()

	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	provider := New(&ConfigYamlImpl{ApiKeyValue: "test-key"})

	baseURL, err := url.Parse(server.URL + "/")
	require.NoError(t, err)

	provider.client.BaseURL = baseURL

	return provider
}

func captureHandler(t *testing.T, requests chan<- resendGo.SendEmailRequest) http.HandlerFunc {
	t.Helper()

	return func(w http.ResponseWriter, r *http.Request) {
		var request resendGo.SendEmailRequest

		assert.NoError(t, json.NewDecoder(r.Body).Decode(&request))

		requests <- request

		w.Header().Set("Content-Type", "application/json")

		_, err := w.Write([]byte(`{"id":"email-123"}`))
		assert.NoError(t, err)
	}
}

func validMessage(body mailer.Body) *mailer.Message {
	return mailer.NewMessage().
		WithFrom(mailer.NewAccount("from@example.com").WithName("Sender")).
		WithTo(mailer.NewAccount("to@example.com")).
		WithCc(mailer.NewAccount("cc@example.com")).
		WithBcc(mailer.NewAccount("bcc@example.com")).
		WithSubject("subject").
		WithBody(body)
}

func TestNew(t *testing.T) {
	t.Parallel()

	provider := New(&ConfigYamlImpl{ApiKeyValue: "test-key"})

	require.NotNil(t, provider)
	require.NotNil(t, provider.client)
}

func TestSendHtmlBody(t *testing.T) {
	t.Parallel()

	requests := make(chan resendGo.SendEmailRequest, 1)
	provider := newTestProvider(t, captureHandler(t, requests))

	result, err := provider.Send(t.Context(), validMessage(mailer.NewHtmlBody("<b>hi</b>")))

	require.NoError(t, err)
	assert.Equal(t, "email-123", result.Id())

	received := <-requests

	assert.Equal(t, "Sender <from@example.com>", received.From)
	assert.Equal(t, []string{"to@example.com"}, received.To)
	assert.Equal(t, []string{"cc@example.com"}, received.Cc)
	assert.Equal(t, []string{"bcc@example.com"}, received.Bcc)
	assert.Equal(t, "subject", received.Subject)
	assert.Equal(t, "<b>hi</b>", received.Html)
	assert.Nil(t, received.Template)
}

func TestSendGoTemplateBody(t *testing.T) {
	t.Parallel()

	requests := make(chan resendGo.SendEmailRequest, 1)
	provider := newTestProvider(t, captureHandler(t, requests))

	body := mailer.NewGoTemplateBody("greeting", "Hello, {{.Name}}!", map[string]any{"Name": "John"})

	_, err := provider.Send(t.Context(), validMessage(body))
	require.NoError(t, err)

	received := <-requests

	assert.Equal(t, "Hello, John!", received.Html)
	assert.Nil(t, received.Template)
}

func TestSendTemplateBody(t *testing.T) {
	t.Parallel()

	requests := make(chan resendGo.SendEmailRequest, 1)
	provider := newTestProvider(t, captureHandler(t, requests))

	body := mailer.NewTemplateBody("tpl-1", map[string]any{"key": "value"})

	result, err := provider.Send(t.Context(), validMessage(body))

	require.NoError(t, err)
	assert.Equal(t, "email-123", result.Id())

	received := <-requests

	require.NotNil(t, received.Template)
	assert.Equal(t, "tpl-1", received.Template.Id)
	assert.Equal(t, map[string]any{"key": "value"}, received.Template.Variables)
	assert.Empty(t, received.Html)
}

func TestSendApiError(t *testing.T) {
	t.Parallel()

	provider := newTestProvider(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)

		_, err := w.Write([]byte(`{"message":"boom"}`))
		assert.NoError(t, err)
	})

	_, err := provider.Send(t.Context(), validMessage(mailer.NewHtmlBody("<b>hi</b>")))
	require.ErrorIs(t, err, errSend)
}

func TestSendContentError(t *testing.T) {
	t.Parallel()

	provider := New(&ConfigYamlImpl{ApiKeyValue: "test-key"})

	body := mailer.NewGoTemplateBody("broken", "{{.Name", nil)

	_, err := provider.Send(t.Context(), validMessage(body))
	require.ErrorIs(t, err, errGetContent)
}
