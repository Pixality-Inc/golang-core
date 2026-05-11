package s3

import (
	"context"
	"errors"
	"net/http"
	"testing"
	"time"

	"github.com/minio/minio-go/v7"
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
			name: "minio NoSuchKey code",
			err:  minio.ErrorResponse{Code: "NoSuchKey", Message: "not found"},
			want: true,
		},
		{
			name: "minio NotFound code (HEAD without body or non-AWS providers)",
			err:  minio.ErrorResponse{Code: "NotFound", Message: "not found"},
			want: true,
		},
		{
			name: "minio http 404 status without code",
			err:  minio.ErrorResponse{StatusCode: http.StatusNotFound, Message: "not found"},
			want: true,
		},
		{
			name: "minio http 500 is not a not-found",
			err:  minio.ErrorResponse{StatusCode: http.StatusInternalServerError, Message: "boom"},
			want: false,
		},
		{
			name: "minio AccessDenied code is not a not-found",
			err:  minio.ErrorResponse{Code: "AccessDenied", StatusCode: http.StatusForbidden, Message: "denied"},
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
	client := NewClient("guard-test", "https://example.invalid", "us-east-1", "", "", "bucket", "" /*baseDir*/, "", true)
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
	client := NewClient("guard-test", "https://example.invalid", "us-east-1", "", "", "bucket", "some-base-dir", "", true)
	t.Cleanup(client.Close)

	err := client.DeleteDir(context.Background(), "")
	require.Error(t, err)
	assert.NotErrorIs(t, err, ErrEmptyDeletePrefix)
}

func TestParseEndpoint(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name       string
		endpoint   string
		wantHost   string
		wantSecure bool
		wantErr    error
	}{
		{
			name:    "empty endpoint is rejected",
			wantErr: ErrEmptyEndpoint,
		},
		{
			name:       "https URL strips scheme and is secure",
			endpoint:   "https://hetzner.example.com",
			wantHost:   "hetzner.example.com",
			wantSecure: true,
		},
		{
			name:       "http URL strips scheme and is not secure",
			endpoint:   "http://localhost:9000",
			wantHost:   "localhost:9000",
			wantSecure: false,
		},
		{
			name:       "bare host defaults to https",
			endpoint:   "hetzner.example.com",
			wantHost:   "hetzner.example.com",
			wantSecure: true,
		},
		{
			name:     "https URL with path is rejected",
			endpoint: "https://host.example.com/minio",
			wantErr:  ErrInvalidEndpoint,
		},
		{
			name:     "https URL with query is rejected",
			endpoint: "https://host.example.com?region=eu",
			wantErr:  ErrInvalidEndpoint,
		},
		{
			name:     "https URL with fragment is rejected",
			endpoint: "https://host.example.com#frag",
			wantErr:  ErrInvalidEndpoint,
		},
		{
			name:     "non-http scheme is rejected",
			endpoint: "ftp://host.example.com",
			wantErr:  ErrInvalidEndpoint,
		},
		{
			name:     "bare host containing a slash is rejected",
			endpoint: "host.example.com/minio",
			wantErr:  ErrInvalidEndpoint,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			host, secure, err := parseEndpoint(testCase.endpoint)

			if testCase.wantErr != nil {
				require.ErrorIs(t, err, testCase.wantErr)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, testCase.wantHost, host)
			assert.Equal(t, testCase.wantSecure, secure)
		})
	}
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

func TestDirEntriesFromObjects(t *testing.T) {
	t.Parallel()

	modTime := time.Unix(1700000000, 0).UTC()

	type entryWant struct {
		name  string
		isDir bool
		size  int64
	}

	cases := []struct {
		name   string
		prefix string
		infos  []minio.ObjectInfo
		want   []entryWant
	}{
		{
			name:   "dirs and files under a nested prefix use tail-only names",
			prefix: "base/dir/",
			infos: []minio.ObjectInfo{
				{Key: "base/dir/subA/"},
				{Key: "base/dir/subB/"},
				{Key: "base/dir/file1.bin", Size: 10, LastModified: modTime},
				{Key: "base/dir/file2.bin", Size: 20, LastModified: modTime},
			},
			want: []entryWant{
				{name: "subA", isDir: true},
				{name: "subB", isDir: true},
				{name: "file1.bin", size: 10},
				{name: "file2.bin", size: 20},
			},
		},
		{
			name:   "zero-byte directory marker duplicating a CommonPrefix is deduped",
			prefix: "base/",
			infos: []minio.ObjectInfo{
				{Key: "base/sub/"},
				{Key: "base/sub/", Size: 0, LastModified: modTime},
				{Key: "base/real.bin", Size: 5, LastModified: modTime},
			},
			want: []entryWant{
				{name: "sub", isDir: true},
				{name: "real.bin", size: 5},
			},
		},
		{
			name:   "the prefix key itself materialized as a zero-byte object is skipped",
			prefix: "base/dir/",
			infos: []minio.ObjectInfo{
				{Key: "base/dir/", Size: 0, LastModified: modTime},
				{Key: "base/dir/keep.bin", Size: 7, LastModified: modTime},
			},
			want: []entryWant{
				{name: "keep.bin", size: 7},
			},
		},
		{
			name:   "empty prefix (bucket root) strips nothing and returns full top-level names",
			prefix: "",
			infos: []minio.ObjectInfo{
				{Key: "top/"},
				{Key: "rootfile.bin", Size: 3, LastModified: modTime},
			},
			want: []entryWant{
				{name: "top", isDir: true},
				{name: "rootfile.bin", size: 3},
			},
		},
		{
			name:   "empty input returns empty slice",
			prefix: "base/",
			infos:  nil,
			want:   nil,
		},
	}

	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			t.Parallel()

			got := dirEntriesFromObjects(testCase.infos, testCase.prefix)

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

// TestDirEntriesFromObjects_UnsortedWithinInputDocuments documents that
// dirEntriesFromObjects itself does NOT sort — ReadDir applies the final
// sort. This guards against accidentally moving the sort into the helper,
// which would look correct in single-input tests but break the assumption
// that ReadDir owns ordering.
func TestDirEntriesFromObjects_UnsortedWithinInput(t *testing.T) {
	t.Parallel()

	got := dirEntriesFromObjects(
		[]minio.ObjectInfo{
			{Key: "base/zdir/"},
			{Key: "base/afile.bin", Size: 1},
		},
		"base/",
	)

	require.Len(t, got, 2)
	assert.Equal(t, "zdir", got[0].Name(), "input order preserved by helper")
	assert.Equal(t, "afile.bin", got[1].Name())
}
