package cache_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pixality-inc/golang-core/cache"
)

var (
	errGetter = errors.New("getter error")
	errSet    = errors.New("set error")
	errHas    = errors.New("has error")
	errFail   = errors.New("fail")
)

type (
	testValue                     string
	fakeCache[K cache.Key, V any] struct {
		hasFn    func(context.Context, K) (bool, error)
		getFn    func(context.Context, K) (V, error)
		setFn    func(context.Context, K, V) error
		defaultV V
		group    cache.Group
	}
)

func (f *fakeCache[K, V]) Has(ctx context.Context, key K) (bool, error) {
	return f.hasFn(ctx, key)
}

func (f *fakeCache[K, V]) Get(ctx context.Context, key K) (V, error) {
	return f.getFn(ctx, key)
}

func (f *fakeCache[K, V]) Set(ctx context.Context, key K, value V) error {
	return f.setFn(ctx, key, value)
}

func (f *fakeCache[K, V]) Default() V {
	return f.defaultV
}

func (f *fakeCache[K, V]) Group() cache.Group {
	return f.group
}

type fakeProxyGetter[V any] struct {
	getFn func(context.Context, cache.Key) (V, error)
}

func (f *fakeProxyGetter[V]) Get(ctx context.Context, key cache.Key) (V, error) {
	return f.getFn(ctx, key)
}

func TestProxy_Get(t *testing.T) {
	t.Parallel()

	defaultValue := testValue("default")

	tests := []struct {
		name    string
		cache   *fakeCache[testKey, testValue]
		getter  *fakeProxyGetter[testValue]
		want    testValue
		wantErr bool
	}{
		{
			name: "cache_has_error",
			cache: &fakeCache[testKey, testValue]{
				hasFn: func(context.Context, testKey) (bool, error) {
					return false, errHas
				},
				defaultV: defaultValue,
			},
			getter:  nil,
			want:    defaultValue,
			wantErr: true,
		},
		{
			name: "cache_hit",
			cache: &fakeCache[testKey, testValue]{
				hasFn: func(context.Context, testKey) (bool, error) {
					return true, nil
				},
				getFn: func(context.Context, testKey) (testValue, error) {
					return "cached", nil
				},
				defaultV: defaultValue,
			},
			getter:  nil,
			want:    "cached",
			wantErr: false,
		},
		{
			name: "cache_miss_getter_success_set_success",
			cache: &fakeCache[testKey, testValue]{
				hasFn: func(context.Context, testKey) (bool, error) {
					return false, nil
				},
				setFn: func(context.Context, testKey, testValue) error {
					return nil
				},
				defaultV: defaultValue,
			},
			getter: &fakeProxyGetter[testValue]{
				getFn: func(ctx context.Context, key cache.Key) (testValue, error) {
					return "from-getter", nil
				},
			},
			want:    "from-getter",
			wantErr: false,
		},
		{
			name: "cache_miss_getter_error",
			cache: &fakeCache[testKey, testValue]{
				hasFn: func(context.Context, testKey) (bool, error) {
					return false, nil
				},
				defaultV: defaultValue,
			},
			getter: &fakeProxyGetter[testValue]{
				getFn: func(ctx context.Context, key cache.Key) (testValue, error) {
					return "", errGetter
				},
			},
			want:    defaultValue,
			wantErr: true,
		},
		{
			name: "cache_miss_getter_success_set_error",
			cache: &fakeCache[testKey, testValue]{
				hasFn: func(context.Context, testKey) (bool, error) {
					return false, nil
				},
				setFn: func(context.Context, testKey, testValue) error {
					return errSet
				},
				defaultV: defaultValue,
			},
			getter: &fakeProxyGetter[testValue]{
				getFn: func(ctx context.Context, key cache.Key) (testValue, error) {
					return "from-getter", nil
				},
			},
			want:    defaultValue,
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			p := cache.NewProxy[testKey, testValue](testCase.cache, testCase.getter)

			val, err := p.Get(context.Background(), testKey("key"))

			if testCase.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			require.Equal(t, testCase.want, val)
		})
	}
}
