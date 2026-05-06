package s3

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	smithyhttp "github.com/aws/smithy-go/transport/http"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var errTestDummy = errors.New("dummy inner transport error")

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

func TestListPrefix(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		fullName string
		want     string
	}{
		{
			name:     "empty stays empty",
			fullName: "",
			want:     "",
		},
		{
			name:     "plain adds trailing slash",
			fullName: "foo/bar",
			want:     "foo/bar/",
		},
		{
			name:     "trailing slash preserved",
			fullName: "foo/bar/",
			want:     "foo/bar/",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.want, listPrefix(tc.fullName))
		})
	}
}

func TestDirEntriesFromPage(t *testing.T) {
	t.Parallel()

	modTime := time.Unix(1700000000, 0).UTC()

	type entryWant struct {
		name  string
		isDir bool
		size  int64
	}

	cases := []struct {
		name           string
		prefix         string
		commonPrefixes []types.CommonPrefix
		contents       []types.Object
		want           []entryWant
	}{
		{
			name:   "dirs and files under a nested prefix use tail-only names",
			prefix: "base/dir/",
			commonPrefixes: []types.CommonPrefix{
				{Prefix: aws.String("base/dir/subA/")},
				{Prefix: aws.String("base/dir/subB/")},
			},
			contents: []types.Object{
				{Key: aws.String("base/dir/file1.bin"), Size: aws.Int64(10), LastModified: aws.Time(modTime)},
				{Key: aws.String("base/dir/file2.bin"), Size: aws.Int64(20), LastModified: aws.Time(modTime)},
			},
			want: []entryWant{
				{name: "subA", isDir: true},
				{name: "subB", isDir: true},
				{name: "file1.bin", size: 10},
				{name: "file2.bin", size: 20},
			},
		},
		{
			name:   "zero-byte directory marker in Contents is skipped (not duplicated against CommonPrefixes)",
			prefix: "base/",
			commonPrefixes: []types.CommonPrefix{
				{Prefix: aws.String("base/sub/")},
			},
			contents: []types.Object{
				{Key: aws.String("base/sub/"), Size: aws.Int64(0), LastModified: aws.Time(modTime)},
				{Key: aws.String("base/real.bin"), Size: aws.Int64(5), LastModified: aws.Time(modTime)},
			},
			want: []entryWant{
				{name: "sub", isDir: true},
				{name: "real.bin", size: 5},
			},
		},
		{
			name:           "the prefix key itself materialized as a zero-byte object is skipped",
			prefix:         "base/dir/",
			commonPrefixes: nil,
			contents: []types.Object{
				{Key: aws.String("base/dir/"), Size: aws.Int64(0), LastModified: aws.Time(modTime)},
				{Key: aws.String("base/dir/keep.bin"), Size: aws.Int64(7), LastModified: aws.Time(modTime)},
			},
			want: []entryWant{
				{name: "keep.bin", size: 7},
			},
		},
		{
			name:   "empty prefix (bucket root) strips nothing and returns full top-level names",
			prefix: "",
			commonPrefixes: []types.CommonPrefix{
				{Prefix: aws.String("top/")},
			},
			contents: []types.Object{
				{Key: aws.String("rootfile.bin"), Size: aws.Int64(3), LastModified: aws.Time(modTime)},
			},
			want: []entryWant{
				{name: "top", isDir: true},
				{name: "rootfile.bin", size: 3},
			},
		},
		{
			name:           "empty page returns empty slice",
			prefix:         "base/",
			commonPrefixes: nil,
			contents:       nil,
			want:           nil,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := dirEntriesFromPage(testCase.commonPrefixes, testCase.contents, testCase.prefix)

			require.Len(t, got, len(testCase.want))

			for i, w := range testCase.want {
				assert.Equal(t, w.name, got[i].Name(), "entry %d name", i)
				assert.Equal(t, w.isDir, got[i].IsDir(), "entry %d isDir", i)

				if !w.isDir {
					info, err := got[i].Info()
					require.NoError(t, err)
					assert.Equal(t, w.size, info.Size(), "entry %d size", i)
				}
			}
		})
	}
}

// TestDirEntriesFromPage_SortedAfterMerge documents that dirEntriesFromPage
// itself does NOT sort — ReadDir applies the final sort across all pages.
// This guards against accidentally moving the sort into the per-page helper,
// which would look correct in single-page tests but break multi-page listings.
func TestDirEntriesFromPage_UnsortedWithinPage(t *testing.T) {
	t.Parallel()

	got := dirEntriesFromPage(
		[]types.CommonPrefix{{Prefix: aws.String("base/zdir/")}},
		[]types.Object{{Key: aws.String("base/afile.bin"), Size: aws.Int64(1)}},
		"base/",
	)

	require.Len(t, got, 2)
	assert.Equal(t, "zdir", got[0].Name(), "dirs appended before files within a page")
	assert.Equal(t, "afile.bin", got[1].Name())
}
