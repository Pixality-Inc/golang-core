package redis

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPubSub_Channel_NilReceiver(t *testing.T) {
	t.Parallel()

	var ps *PubSub

	ch := ps.Channel()
	require.Nil(t, ch, "Channel() on nil PubSub should return nil")
}

func TestPubSub_Channel_NilInner(t *testing.T) {
	t.Parallel()

	ps := &PubSub{pubsub: nil}

	ch := ps.Channel()
	require.Nil(t, ch, "Channel() with nil inner pubsub should return nil")
}

func TestPubSub_Close_NilReceiver(t *testing.T) {
	t.Parallel()

	var ps *PubSub

	err := ps.Close()
	require.NoError(t, err, "Close() on nil PubSub should not error")
}

func TestPubSub_Close_NilInner(t *testing.T) {
	t.Parallel()

	ps := &PubSub{pubsub: nil}

	err := ps.Close()
	require.NoError(t, err, "Close() with nil inner pubsub should not error")
}

func TestPubSub_Subscribe_NotConnected(t *testing.T) {
	t.Parallel()

	var ps *PubSub

	require.ErrorIs(t, ps.Subscribe(t.Context(), "room"), ErrPubSubNotConnected)
	require.ErrorIs(t, (&PubSub{}).Subscribe(t.Context(), "room"), ErrPubSubNotConnected)
}

func TestPubSub_Unsubscribe_NotConnected(t *testing.T) {
	t.Parallel()

	var ps *PubSub

	require.ErrorIs(t, ps.Unsubscribe(t.Context(), "room"), ErrPubSubNotConnected)
	require.ErrorIs(t, (&PubSub{}).Unsubscribe(t.Context(), "room"), ErrPubSubNotConnected)
}

func TestPubSub_ChannelWithOptions_NotConnected(t *testing.T) {
	t.Parallel()

	var ps *PubSub

	require.Nil(t, ps.ChannelWithOptions(WithChannelSize(10)))
}
