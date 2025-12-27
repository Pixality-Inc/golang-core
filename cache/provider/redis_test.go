package provider_test

import (
	"context"
	"errors"
	"testing"
	"time"

	goredis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/pixality-inc/golang-core/cache/provider"
	redisMock "github.com/pixality-inc/golang-core/redis/mocks"
)

var errFail = errors.New("fail")

func TestRedis_Has(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		mockSetup func(m *redisMock.MockClient)
		want      bool
		wantErr   bool
	}{
		{
			name: "key_exists",
			mockSetup: func(m *redisMock.MockClient) {
				m.EXPECT().
					GetString(gomock.Any(), "grp:key").
					Return("value", nil)
			},
			want: true,
		},
		{
			name: "key_missing",
			mockSetup: func(m *redisMock.MockClient) {
				m.EXPECT().
					GetString(gomock.Any(), "grp:key").
					Return("", goredis.Nil)
			},
			want: false,
		},
		{
			name: "redis_error",
			mockSetup: func(m *redisMock.MockClient) {
				m.EXPECT().
					GetString(gomock.Any(), "grp:key").
					Return("", errFail)
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockClient := redisMock.NewMockClient(ctrl)
			if testCase.mockSetup != nil {
				testCase.mockSetup(mockClient)
			}

			r := provider.NewRedis(mockClient)

			ok, err := r.Has(context.Background(), "grp", "key")

			if testCase.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			require.Equal(t, testCase.want, ok)
		})
	}
}

func TestRedis_Get(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		mockSetup func(m *redisMock.MockClient)
		want      []byte
		wantErr   bool
	}{
		{
			name: "key_exists",
			mockSetup: func(m *redisMock.MockClient) {
				m.EXPECT().
					GetString(gomock.Any(), "grp:key").
					Return("value", nil)
			},
			want: []byte("value"),
		},
		{
			name: "key_missing",
			mockSetup: func(m *redisMock.MockClient) {
				m.EXPECT().
					GetString(gomock.Any(), "grp:key").
					Return("", goredis.Nil)
			},
			wantErr: true,
		},
		{
			name: "redis_error",
			mockSetup: func(m *redisMock.MockClient) {
				m.EXPECT().
					GetString(gomock.Any(), "grp:key").
					Return("", errFail)
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockClient := redisMock.NewMockClient(ctrl)
			if testCase.mockSetup != nil {
				testCase.mockSetup(mockClient)
			}

			r := provider.NewRedis(mockClient)

			val, err := r.Get(context.Background(), "grp", "key")

			if testCase.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			require.Equal(t, testCase.want, val)
		})
	}
}

func TestRedis_Set(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		mockSetup func(m *redisMock.MockClient)
		wantErr   bool
	}{
		{
			name: "success",
			mockSetup: func(m *redisMock.MockClient) {
				m.EXPECT().
					SetKey(gomock.Any(), "grp:key", "value", time.Second).
					Return(nil)
			},
		},
		{
			name: "redis_error",
			mockSetup: func(m *redisMock.MockClient) {
				m.EXPECT().
					SetKey(gomock.Any(), "grp:key", "value", time.Second).
					Return(errFail)
			},
			wantErr: true,
		},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			mockClient := redisMock.NewMockClient(ctrl)
			if testCase.mockSetup != nil {
				testCase.mockSetup(mockClient)
			}

			r := provider.NewRedis(mockClient)

			err := r.Set(context.Background(), "grp", "key", []byte("value"), time.Second)

			if testCase.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
		})
	}
}
