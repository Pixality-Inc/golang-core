package cache_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/pixality-inc/golang-core/cache"
	"github.com/pixality-inc/golang-core/cache/marshal"
	"github.com/pixality-inc/golang-core/cache/provider"
)

type testStruct struct {
	Simple any
}

func TestCacheIntegration_Get(t *testing.T) {
	t.Parallel()

	defaultVal := testStruct{Simple: "hello"}

	tests := []struct {
		name        string
		providerSet func(p *provider.Memory)
		want        testStruct
		wantErr     error
	}{
		{
			name:        "no_such_key_returns_default",
			providerSet: func(_ *provider.Memory) {},
			want:        defaultVal,
		},
		{
			name: "unmarshal_error",
			providerSet: func(p *provider.Memory) {
				err := p.Set(t.Context(), "grp", "key", []byte(`{`), time.Minute)
				require.NoError(t, err)
			},
			wantErr: cache.ErrUnmarshal,
		},
		{
			name: "success",
			providerSet: func(p *provider.Memory) {
				err := p.Set(t.Context(), "grp", "key", []byte(`{"simple":"json"}`), time.Minute)
				require.NoError(t, err)
			},
			want: testStruct{Simple: "json"},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			memProvider := provider.NewMemory()
			testCase.providerSet(memProvider)

			marshaller := marshal.NewJsonMarshaller()

			testCache := cache.NewCache[testKey, testStruct](
				"grp",
				marshaller,
				memProvider,
				defaultVal,
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

func TestCacheIntegration_Set(t *testing.T) {
	t.Parallel()

	defaultVal := testStruct{Simple: "hello"}

	tests := []struct {
		name    string
		value   testStruct
		wantErr error
	}{
		{
			name:    "marshal_error",
			value:   testStruct{Simple: make(chan struct{})},
			wantErr: cache.ErrMarshal,
		},
		{
			name:  "success",
			value: testStruct{Simple: "json"},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			marshaller := marshal.NewJsonMarshaller()

			memProvider := provider.NewMemory()

			testCache := cache.NewCache[testKey, testStruct](
				"grp",
				marshaller,
				memProvider,
				defaultVal,
				time.Second,
			)

			err := testCache.Set(context.Background(), "key", testCase.value)

			if testCase.wantErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, testCase.wantErr)

				return
			}

			require.NoError(t, err)
		})
	}
}

func TestCacheIntegration_Has(t *testing.T) {
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

func TestCacheIntegration_DefaultAndGroup(t *testing.T) {
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
