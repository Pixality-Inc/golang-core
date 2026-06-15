package redis_test

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/pixality-inc/golang-core/redis"
	redisMocks "github.com/pixality-inc/golang-core/redis/mocks"
)

var errRedis = errors.New("redis failed")

type testEntity struct {
	Name string `json:"name"`
}

func TestSet(t *testing.T) {
	t.Parallel()

	client := redisMocks.NewMockClient(gomock.NewController(t))
	client.EXPECT().SetKey(gomock.Any(), "user:1", `{"name":"John"}`, time.Minute).Return(nil)

	err := redis.Set(t.Context(), client, "user:1", testEntity{Name: "John"}, time.Minute)
	require.NoError(t, err)
}

func TestSetMarshalError(t *testing.T) {
	t.Parallel()

	client := redisMocks.NewMockClient(gomock.NewController(t))

	err := redis.Set(t.Context(), client, "key", make(chan int), time.Minute)
	require.Error(t, err)
}

func TestSetClientError(t *testing.T) {
	t.Parallel()

	client := redisMocks.NewMockClient(gomock.NewController(t))
	client.EXPECT().SetKey(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(errRedis)

	err := redis.Set(t.Context(), client, "key", testEntity{}, time.Minute)
	require.ErrorIs(t, err, errRedis)
}

func TestGet(t *testing.T) {
	t.Parallel()

	client := redisMocks.NewMockClient(gomock.NewController(t))
	client.EXPECT().GetString(gomock.Any(), "user:1").Return(`{"name":"John"}`, nil)

	value, err := redis.Get(t.Context(), client, "user:1", testEntity{})
	require.NoError(t, err)
	assert.Equal(t, testEntity{Name: "John"}, value)
}

func TestGetClientError(t *testing.T) {
	t.Parallel()

	client := redisMocks.NewMockClient(gomock.NewController(t))
	client.EXPECT().GetString(gomock.Any(), gomock.Any()).Return("", errRedis)

	defaultValue := testEntity{Name: "default"}

	value, err := redis.Get(t.Context(), client, "key", defaultValue)
	require.ErrorIs(t, err, errRedis)
	assert.Equal(t, defaultValue, value)
}

func TestGetInvalidJson(t *testing.T) {
	t.Parallel()

	client := redisMocks.NewMockClient(gomock.NewController(t))
	client.EXPECT().GetString(gomock.Any(), gomock.Any()).Return("not json", nil)

	defaultValue := testEntity{Name: "default"}

	value, err := redis.Get(t.Context(), client, "key", defaultValue)
	require.Error(t, err)
	assert.Equal(t, defaultValue, value)
}

func TestPublish(t *testing.T) {
	t.Parallel()

	client := redisMocks.NewMockClient(gomock.NewController(t))
	client.EXPECT().Publish(gomock.Any(), "events", `{"name":"John"}`).Return(nil)

	err := redis.Publish(t.Context(), client, "events", testEntity{Name: "John"})
	require.NoError(t, err)
}

func TestPublishMarshalError(t *testing.T) {
	t.Parallel()

	client := redisMocks.NewMockClient(gomock.NewController(t))

	err := redis.Publish(t.Context(), client, "events", make(chan int))
	require.Error(t, err)
}

func TestPublishClientError(t *testing.T) {
	t.Parallel()

	client := redisMocks.NewMockClient(gomock.NewController(t))
	client.EXPECT().Publish(gomock.Any(), gomock.Any(), gomock.Any()).Return(errRedis)

	err := redis.Publish(t.Context(), client, "events", testEntity{})
	require.ErrorIs(t, err, errRedis)
}

func TestSubscribeNilPubSub(t *testing.T) {
	t.Parallel()

	client := redisMocks.NewMockClient(gomock.NewController(t))
	client.EXPECT().Subscribe(gomock.Any(), "events").Return(nil)

	values, closeFn := redis.Subscribe[testEntity](t.Context(), client, "events")

	assert.Nil(t, values)
	require.NoError(t, closeFn())
}
