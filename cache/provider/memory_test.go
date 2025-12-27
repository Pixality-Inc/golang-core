package provider_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/pixality-inc/golang-core/cache"
	"github.com/pixality-inc/golang-core/cache/provider"
)

func TestMemory_Set_Get(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		ttl     time.Duration
		wantErr error
	}{
		{
			name: "value_exists",
			ttl:  time.Second,
		},
		{
			name:    "expired_value",
			ttl:     -time.Second,
			wantErr: cache.ErrProviderNoSuchKey,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mem := provider.NewMemory()
			ctx := context.Background()

			err := mem.Set(ctx, "grp", "key", []byte("value"), testCase.ttl)
			require.NoError(t, err)

			val, err := mem.Get(ctx, "grp", "key")

			if testCase.wantErr != nil {
				require.Error(t, err)
				require.ErrorIs(t, err, testCase.wantErr)

				return
			}

			require.NoError(t, err)
			require.Equal(t, []byte("value"), val)
		})
	}
}

func TestMemory_Get_NoSuchKey(t *testing.T) {
	t.Parallel()

	mem := provider.NewMemory()

	val, err := mem.Get(context.Background(), "grp", "missing")

	require.Nil(t, val)
	require.Error(t, err)
	require.ErrorIs(t, err, cache.ErrProviderNoSuchKey)
}

func TestMemory_Has(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		setup func(m *provider.Memory)
		want  bool
	}{
		{
			name: "key_exists",
			setup: func(m *provider.Memory) {
				require.NoError(t,
					m.Set(context.Background(), "grp", "key", []byte("v"), time.Second),
				)
			},
			want: true,
		},
		{
			name:  "key_missing",
			setup: func(_ *provider.Memory) {},
			want:  false,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			mem := provider.NewMemory()
			ctx := context.Background()

			if testCase.setup != nil {
				testCase.setup(mem)
			}

			ok, err := mem.Has(ctx, "grp", "key")

			require.NoError(t, err)
			require.Equal(t, testCase.want, ok)
		})
	}
}

func TestMemory_Get_ExpiredDeletesEntry(t *testing.T) {
	t.Parallel()

	mem := provider.NewMemory()
	ctx := context.Background()

	require.NoError(t,
		mem.Set(ctx, "grp", "key", []byte("v"), -time.Second),
	)

	_, err := mem.Get(ctx, "grp", "key")
	require.ErrorIs(t, err, cache.ErrProviderNoSuchKey)

	ok, err := mem.Has(ctx, "grp", "key")
	require.NoError(t, err)
	require.False(t, ok, "expired entry must be deleted")
}
