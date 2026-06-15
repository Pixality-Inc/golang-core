package kafka

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

var errPing = errors.New("ping failed")

type fakePingable struct {
	connected bool
	pingErr   error
}

func (f *fakePingable) IsConnected() bool {
	return f.connected
}

func (f *fakePingable) Ping(_ context.Context) error {
	return f.pingErr
}

type blockingPingable struct{}

func (b *blockingPingable) IsConnected() bool {
	return true
}

func (b *blockingPingable) Ping(ctx context.Context) error {
	<-ctx.Done()

	return ctx.Err()
}

func TestHealthcheckServiceNotConnected(t *testing.T) {
	t.Parallel()

	service := NewHealthcheckService(&fakePingable{connected: false, pingErr: errPing})

	assert.True(t, service.IsOK())
}

func TestHealthcheckServiceConnectedHealthy(t *testing.T) {
	t.Parallel()

	service := NewHealthcheckService(&fakePingable{connected: true})

	assert.True(t, service.IsOK())
}

func TestHealthcheckServiceConnectedPingFails(t *testing.T) {
	t.Parallel()

	service := NewHealthcheckService(&fakePingable{connected: true, pingErr: errPing})

	assert.False(t, service.IsOK())
}

func TestHealthcheckServiceTimeoutApplied(t *testing.T) {
	t.Parallel()

	service := NewHealthcheckService(&blockingPingable{}, WithHealthcheckTimeout(10*time.Millisecond))

	start := time.Now()

	assert.False(t, service.IsOK())
	assert.Less(t, time.Since(start), time.Second)
}

func TestHealthcheckServiceDefaultTimeout(t *testing.T) {
	t.Parallel()

	service := NewHealthcheckService(&fakePingable{})

	assert.Equal(t, defaultHealthcheckTimeout, service.timeout)
}

func TestTLSConfigYaml(t *testing.T) {
	t.Parallel()

	config := &TLSConfigYaml{
		EnabledValue:  true,
		CAFileValue:   "/ca.pem",
		CertFileValue: "/cert.pem",
		KeyFileValue:  "/key.pem",
	}

	assert.True(t, config.Enabled())
	assert.Equal(t, "/ca.pem", config.CAFile())
	assert.Equal(t, "/cert.pem", config.CertFile())
	assert.Equal(t, "/key.pem", config.KeyFile())

	empty := &TLSConfigYaml{}

	assert.False(t, empty.Enabled())
	assert.Empty(t, empty.CAFile())
	assert.Empty(t, empty.CertFile())
	assert.Empty(t, empty.KeyFile())
}
