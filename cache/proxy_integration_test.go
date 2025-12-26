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

func TestProxyIntegration_Get(t *testing.T) {
	t.Parallel()

	defaultValue := testStruct{Simple: "hello"}

	tests := []struct {
		name    string
		cache   func() cache.Cache[testKey, testStruct]
		getter  *fakeProxyGetter[testStruct]
		want    testStruct
		wantErr bool
	}{
		{
			name: "cache_hit",
			cache: func() cache.Cache[testKey, testStruct] {
				memProvider := provider.NewMemory()
				err := memProvider.Set(t.Context(), "grp", "key", []byte(`{"simple":"json"}`), time.Minute)
				require.NoError(t, err)

				marshaller := marshal.NewJsonMarshaller()

				return cache.NewCache[testKey, testStruct](
					"grp",
					marshaller,
					memProvider,
					defaultValue,
					time.Second,
				)
			},
			getter: nil,
			want:   testStruct{Simple: "json"},
		},
		{
			name: "cache_miss",
			cache: func() cache.Cache[testKey, testStruct] {
				memProvider := provider.NewMemory()

				marshaller := marshal.NewJsonMarshaller()

				return cache.NewCache[testKey, testStruct](
					"grp",
					marshaller,
					memProvider,
					defaultValue,
					time.Second,
				)
			},
			getter: &fakeProxyGetter[testStruct]{
				getFn: func(ctx context.Context, key cache.Key) (testStruct, error) {
					return testStruct{Simple: "done"}, nil
				},
			},
			want: testStruct{Simple: "done"},
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			proxy := cache.NewProxy[testKey, testStruct](testCase.cache(), testCase.getter)

			val, err := proxy.Get(context.Background(), testKey("key"))

			if testCase.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, testCase.want, val)
		})
	}
}
