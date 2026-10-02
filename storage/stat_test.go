package storage

import (
	"context"
	"errors"
	"io/fs"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

type statProvider struct {
	Provider

	info   fs.FileInfo
	errs   []error
	calls  int
	onCall func(context.Context, string)
}

func (p *statProvider) Stat(ctx context.Context, path string) (fs.FileInfo, error) {
	p.onCall(ctx, path)

	index := p.calls
	p.calls++

	if index < len(p.errs) && p.errs[index] != nil {
		return nil, p.errs[index]
	}

	return p.info, nil
}

func TestStorageStat(t *testing.T) {
	t.Parallel()

	want, err := NewFileEntry("object.bin", 123, time.Unix(1700000000, 0)).Info()
	require.NoError(t, err)

	cases := []struct {
		name      string
		errs      []error
		retry     bool
		wantErr   error
		wantCalls int
	}{
		{name: "success", wantCalls: 1},
		{name: "no retry by default", errs: []error{errTransient}, wantErr: errTransient, wantCalls: 1},
		{name: "transient error retried", errs: []error{errTransient}, retry: true, wantCalls: 2},
		{name: "retry exhausted", errs: []error{errTransient, errTransient, errTransient}, retry: true, wantErr: errTransient, wantCalls: 3},
		{name: "missing object", errs: []error{fs.ErrNotExist}, retry: true, wantErr: fs.ErrNotExist, wantCalls: 1},
		{name: "permission denied", errs: []error{fs.ErrPermission}, retry: true, wantErr: fs.ErrPermission, wantCalls: 1},
		{name: "canceled", errs: []error{context.Canceled}, retry: true, wantErr: context.Canceled, wantCalls: 1},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			ctx := t.Context()

			provider := &statProvider{
				info: want,
				errs: testCase.errs,
				onCall: func(actualCtx context.Context, path string) {
					require.Same(t, ctx, actualCtx)
					require.Equal(t, "nested/object.bin", path)
				},
			}

			var opts []Option
			if testCase.retry {
				opts = append(opts, WithRetry(enabledPolicy()))
			}

			store := NewStorage(provider, nil, opts...)

			info, err := store.Stat(ctx, "nested/object.bin")
			if testCase.wantErr == nil {
				require.NoError(t, err)
				require.Same(t, want, info)
			} else {
				require.ErrorIs(t, err, testCase.wantErr)
				require.ErrorContains(t, err, "storage.Stat(nested/object.bin)")
				require.Nil(t, info)

				if errors.Is(testCase.wantErr, fs.ErrNotExist) {
					require.True(t, IsNotFound(err))
				}
			}

			require.Equal(t, testCase.wantCalls, provider.calls)
		})
	}
}
