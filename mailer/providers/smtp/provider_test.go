package smtp_test

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pixality-inc/golang-core/mailer"
	"github.com/pixality-inc/golang-core/mailer/providers/smtp"
)

func newConfig(host string, port int) *smtp.ConfigYamlImpl {
	return &smtp.ConfigYamlImpl{
		HostValue: host,
		PortValue: port,
	}
}

func newProvider(t *testing.T) *smtp.Smtp {
	t.Helper()

	provider, err := smtp.New(newConfig("localhost", 25))
	require.NoError(t, err)

	return provider
}

func newRefusingServerPort(t *testing.T) int {
	t.Helper()

	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "127.0.0.1:0")
	require.NoError(t, err)

	t.Cleanup(func() {
		if closeErr := listener.Close(); closeErr != nil {
			t.Logf("closing listener: %v", closeErr)
		}
	})

	go func() {
		for {
			conn, acceptErr := listener.Accept()
			if acceptErr != nil {
				return
			}

			if closeErr := conn.Close(); closeErr != nil {
				return
			}
		}
	}()

	addr, ok := listener.Addr().(*net.TCPAddr)
	require.True(t, ok)

	return addr.Port
}

func validMessage(body mailer.Body) *mailer.Message {
	return mailer.NewMessage().
		WithFrom(mailer.NewAccount("from@example.com")).
		WithTo(mailer.NewAccount("to@example.com")).
		WithSubject("subject").
		WithBody(body)
}

func TestNewWithoutAuth(t *testing.T) {
	t.Parallel()

	provider, err := smtp.New(newConfig("localhost", 25))
	require.NoError(t, err)
	require.NotNil(t, provider)
}

func TestNewWithAuth(t *testing.T) {
	t.Parallel()

	config := newConfig("localhost", 587)
	config.UsernameValue = "user"
	config.PasswordValue = "pass"

	provider, err := smtp.New(config)
	require.NoError(t, err)
	require.NotNil(t, provider)
}

func TestNewEmptyHost(t *testing.T) {
	t.Parallel()

	_, err := smtp.New(newConfig("", 25))
	require.ErrorContains(t, err, "client")
}

func TestSendTemplateBodyNotSupported(t *testing.T) {
	t.Parallel()

	_, err := newProvider(t).Send(t.Context(), validMessage(mailer.NewTemplateBody("tpl", nil)))
	require.ErrorContains(t, err, "template not supported")
}

func TestSendContentError(t *testing.T) {
	t.Parallel()

	_, err := newProvider(t).Send(t.Context(), validMessage(mailer.NewGoTemplateBody("broken", "{{.Name", nil)))
	require.ErrorContains(t, err, "get content")
}

func TestSendInvalidAddress(t *testing.T) {
	t.Parallel()

	testCases := []struct {
		name   string
		mutate func(message *mailer.Message)
	}{
		{"from", func(message *mailer.Message) { message.From = mailer.NewAccount("not an email") }},
		{"to", func(message *mailer.Message) { message.To = []mailer.Account{mailer.NewAccount("not an email")} }},
		{"cc", func(message *mailer.Message) { message.Cc = []mailer.Account{mailer.NewAccount("not an email")} }},
		{"bcc", func(message *mailer.Message) { message.Bcc = []mailer.Account{mailer.NewAccount("not an email")} }},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			message := validMessage(mailer.NewTextBody("body"))
			testCase.mutate(message)

			_, err := newProvider(t).Send(t.Context(), message)
			require.Error(t, err)
		})
	}
}

func TestSendDialError(t *testing.T) {
	t.Parallel()

	provider, err := smtp.New(newConfig("127.0.0.1", newRefusingServerPort(t)))
	require.NoError(t, err)

	_, err = provider.Send(t.Context(), validMessage(mailer.NewTextBody("body")))
	require.ErrorContains(t, err, "send")
}
