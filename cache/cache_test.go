package cache_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/pixality-inc/golang-core/cache"
	"github.com/pixality-inc/golang-core/cache/provider"
)

type testKey string

func (k testKey) String() string {
	return string(k)
}

type fakeMarshaller struct {
	marshalFn   func(v any) ([]byte, error)
	unmarshalFn func(data []byte, v any) error
}

func (f fakeMarshaller) Marshal(v any) ([]byte, error) {
	return f.marshalFn(v)
}

func (f fakeMarshaller) Unmarshal(data []byte, v any) error {
	return f.unmarshalFn(data, v)
}

type fakeProvider struct {
	hasFn func(ctx context.Context, group cache.Group, key string) (bool, error)
	getFn func(ctx context.Context, group cache.Group, key string) ([]byte, error)
	setFn func(ctx context.Context, group cache.Group, key string, value []byte, ttl time.Duration) error
}

func (f fakeProvider) Has(ctx context.Context, group cache.Group, key string) (bool, error) {
	return f.hasFn(ctx, group, key)
}

func (f fakeProvider) Get(ctx context.Context, group cache.Group, key string) ([]byte, error) {
	return f.getFn(ctx, group, key)
}

func (f fakeProvider) Set(ctx context.Context, group cache.Group, key string, value []byte, ttl time.Duration) error {
	return f.setFn(ctx, group, key, value, ttl)
}

func TestCache_Get(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		providerGet func() ([]byte, error)
		unmarshal   func([]byte, any) error
		defaultVal  int
		want        int
		wantErr     error
	}{
		{
			name:       "no_such_key_returns_default",
			defaultVal: 42,
			providerGet: func() ([]byte, error) {
				return nil, cache.ErrProviderNoSuchKey
			},
			want: 42,
		},
		{
			name:       "provider_error",
			defaultVal: 0,
			providerGet: func() ([]byte, error) {
				return nil, errFail
			},
			wantErr: cache.ErrProviderGet,
		},
		{
			name:       "unmarshal_error",
			defaultVal: 0,
			providerGet: func() ([]byte, error) {
				return []byte("bad"), nil
			},
			unmarshal: func([]byte, any) error {
				return cache.ErrUnmarshal
			},
			wantErr: cache.ErrUnmarshal,
		},
		{
			name:       "success",
			defaultVal: 0,
			providerGet: func() ([]byte, error) {
				return []byte("42"), nil
			},
			unmarshal: func(_ []byte, v any) error {
				*(v.(*int)) = 42 // nolint:errcheck,forcetypeassert

				return nil
			},
			want: 42,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			prov := &fakeProvider{
				getFn: func(ctx context.Context, group cache.Group, key string) ([]byte, error) {
					require.Equal(t, cache.Group("grp"), group)
					require.Equal(t, "key", key)

					return testCase.providerGet()
				},
			}

			marshaller := &fakeMarshaller{
				unmarshalFn: testCase.unmarshal,
			}

			testCache := cache.NewCache[testKey, int](
				"grp",
				marshaller,
				prov,
				testCase.defaultVal,
				time.Second,
			)

			got, err := testCache.Get(context.Background(), "key")

			if testCase.wantErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, testCase.wantErr)

				return
			}

			require.NoError(t, err)
			require.Equal(t, testCase.want, got)
		})
	}
}

func TestCache_Set(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		marshal func(any) ([]byte, error)
		set     func(b []byte) error
		wantErr error
	}{
		{
			name: "marshal_error",
			marshal: func(any) ([]byte, error) {
				return nil, errFail
			},
			wantErr: cache.ErrMarshal,
		},
		{
			name: "provider_error",
			marshal: func(any) ([]byte, error) {
				return []byte("1"), nil
			},
			set: func(b []byte) error {
				return errSet
			},
			wantErr: cache.ErrProviderSet,
		},
		{
			name: "success",
			marshal: func(any) ([]byte, error) {
				return []byte("1"), nil
			},
			set: func(b []byte) error {
				require.Equal(t, []byte("1"), b)

				return nil
			},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			marshaler := &fakeMarshaller{
				marshalFn: testCase.marshal,
			}
			prov := &fakeProvider{
				setFn: func(ctx context.Context, group cache.Group, key string, value []byte, ttl time.Duration) error {
					require.Equal(t, cache.Group("grp"), group)
					require.Equal(t, "key", key)

					return testCase.set(value)
				},
			}

			testCache := cache.NewCache[testKey, int](
				"grp",
				marshaler,
				prov,
				0,
				time.Second,
			)

			err := testCache.Set(context.Background(), "key", 1)

			if testCase.wantErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, testCase.wantErr)

				return
			}

			require.NoError(t, err)
		})
	}
}

func TestCache_Has(t *testing.T) {
	t.Parallel()

	memProvider := provider.NewMemory()
	err := memProvider.Set(t.Context(), "grp", "key", []byte{}, time.Second)
	require.NoError(t, err)

	testCache := cache.NewCache[testKey, int](
		"grp",
		nil,
		memProvider,
		0,
		0,
	)

	ok, err := testCache.Has(context.Background(), "key")
	require.NoError(t, err)
	require.True(t, ok)
}

func TestCache_DefaultAndGroup(t *testing.T) {
	t.Parallel()

	testCache := cache.NewCache[testKey, int](
		"grp",
		nil,
		provider.NewMemory(),
		42,
		0,
	)

	require.Equal(t, 42, testCache.Default())
	require.Equal(t, cache.Group("grp"), testCache.Group())
}
