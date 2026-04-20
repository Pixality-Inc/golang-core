package s3

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errTestDummy = errors.New("dummy inner transport error")

func TestBuildCopySource(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name   string
		bucket string
		key    string
		want   string
	}{
		{
			name:   "plain ascii with slashes preserved",
			bucket: "my-bucket",
			key:    "dir/sub/file.bin",
			want:   "my-bucket/dir/sub/file.bin",
		},
		{
			name:   "unicode key is percent-encoded, slashes preserved",
			bucket: "my-bucket",
			key:    "dir/файл.bin",
			want:   "my-bucket/dir/%D1%84%D0%B0%D0%B9%D0%BB.bin",
		},
		{
			name:   "reserved chars are percent-encoded",
			bucket: "my-bucket",
			key:    "a b/c?d#e.bin",
			want:   "my-bucket/a%20b/c%3Fd%23e.bin",
		},
		{
			name:   "empty base dir key",
			bucket: "my-bucket",
			key:    "file.bin",
			want:   "my-bucket/file.bin",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, buildCopySource(tc.bucket, tc.key))
		})
	}
}

func TestIsNotFoundErr(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "aws typed NotFound",
			err:  &types.NotFound{},
			want: true,
		},
		{
			name: "smithy http 404",
			err: &smithyhttp.ResponseError{
				Response: &smithyhttp.Response{Response: &http.Response{StatusCode: http.StatusNotFound}},
				Err:      errTestDummy,
			},
			want: true,
		},
		{
			name: "smithy http 500 is not a not-found",
			err: &smithyhttp.ResponseError{
				Response: &smithyhttp.Response{Response: &http.Response{StatusCode: http.StatusInternalServerError}},
				Err:      errTestDummy,
			},
			want: false,
		},
		{
			name: "smithy api error code NotFound",
			err:  &smithy.GenericAPIError{Code: "NotFound", Message: "not found"},
			want: true,
		},
		{
			name: "smithy api error code NoSuchKey",
			err:  &smithy.GenericAPIError{Code: "NoSuchKey", Message: "no such key"},
			want: true,
		},
		{
			name: "smithy api error with unrelated code",
			err:  &smithy.GenericAPIError{Code: "AccessDenied", Message: "denied"},
			want: false,
		},
		{
			name: "plain error",
			err:  errTestDummy,
			want: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, isNotFoundErr(tc.err))
		})
	}
}

func TestDeleteDir_EmptyPrefixGuard(t *testing.T) {
	t.Parallel()

	// Empty baseDir + empty objectName must be rejected BEFORE init runs,
	// so no creds / network are needed to reach the guard.
	client := NewClient("guard-test", "", "us-east-1", "", "", "bucket", "" /*baseDir*/, "", true)
	t.Cleanup(client.Close)

	err := client.DeleteDir(context.Background(), "")
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEmptyDeletePrefix)
}

func TestDeleteDir_EmptyObjectNameUnderBaseDirIsAllowed(t *testing.T) {
	t.Parallel()

	// With a baseDir set, DeleteDir(ctx, "") is a legitimate "wipe my subtree"
	// call. The guard must NOT trip here — it should proceed to init, which
	// will then fail with an auth/config error (good enough proof the guard
	// didn't short-circuit).
	client := NewClient("guard-test", "", "us-east-1", "", "", "bucket", "some-base-dir", "", true)
	t.Cleanup(client.Close)

	err := client.DeleteDir(context.Background(), "")
	require.Error(t, err)
	assert.NotErrorIs(t, err, ErrEmptyDeletePrefix)
}
