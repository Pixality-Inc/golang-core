package gcs

import (
	"io/fs"
	"testing"
	"time"

	gcs "cloud.google.com/go/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func TestDirEntry_File(t *testing.T) {
	t.Parallel()

	now := time.Unix(1700000000, 0).UTC()
	entry := &dirEntry{name: "video.mp4", size: 1024, modTime: now}

	assert.Equal(t, "video.mp4", entry.Name())
	assert.False(t, entry.IsDir())
	assert.Equal(t, fs.FileMode(0), entry.Mode())
	assert.Equal(t, fs.FileMode(0), entry.Type())
	assert.Equal(t, int64(1024), entry.Size())
	assert.Equal(t, now, entry.ModTime())
	assert.Nil(t, entry.Sys())

	info, err := entry.Info()
	require.NoError(t, err)
	assert.Equal(t, "video.mp4", info.Name())
	assert.Equal(t, int64(1024), info.Size())
	assert.False(t, info.IsDir())
}

func TestDirEntry_Dir(t *testing.T) {
	t.Parallel()

	entry := &dirEntry{name: "subdir", isDir: true}

	assert.Equal(t, "subdir", entry.Name())
	assert.True(t, entry.IsDir())
	assert.Equal(t, fs.ModeDir, entry.Mode())
	assert.Equal(t, fs.ModeDir, entry.Type())
	assert.Equal(t, int64(0), entry.Size())
	assert.True(t, entry.ModTime().IsZero())

	info, err := entry.Info()
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestDirEntryFromAttrs(t *testing.T) {
	t.Parallel()

	modTime := time.Unix(1700000000, 0).UTC()

	cases := []struct {
		name      string
		attrs     *gcs.ObjectAttrs
		prefix    string
		wantNil   bool
		wantName  string
		wantIsDir bool
		wantSize  int64
	}{
		{
			name:      "common prefix becomes dir entry with tail-only name",
			attrs:     &gcs.ObjectAttrs{Prefix: "base/dir/sub/"},
			prefix:    "base/dir/",
			wantName:  "sub",
			wantIsDir: true,
		},
		{
			name:     "object name becomes file entry with tail-only name",
			attrs:    &gcs.ObjectAttrs{Name: "base/dir/video.mp4", Size: 1024, Updated: modTime},
			prefix:   "base/dir/",
			wantName: "video.mp4",
			wantSize: 1024,
		},
		{
			name:    "zero-byte directory marker in Name is skipped",
			attrs:   &gcs.ObjectAttrs{Name: "base/dir/sub/", Size: 0, Updated: modTime},
			prefix:  "base/dir/",
			wantNil: true,
		},
		{
			name:    "the prefix itself materialized as a zero-byte object is skipped",
			attrs:   &gcs.ObjectAttrs{Name: "base/dir/", Size: 0, Updated: modTime},
			prefix:  "base/dir/",
			wantNil: true,
		},
		{
			name:    "nil attrs returns nil",
			attrs:   nil,
			prefix:  "base/dir/",
			wantNil: true,
		},
		{
			name:     "empty prefix (bucket root) strips nothing",
			attrs:    &gcs.ObjectAttrs{Name: "rootfile.bin", Size: 3, Updated: modTime},
			prefix:   "",
			wantName: "rootfile.bin",
			wantSize: 3,
		},
		{
			name:      "empty prefix with common prefix returns top-level dir name",
			attrs:     &gcs.ObjectAttrs{Prefix: "top/"},
			prefix:    "",
			wantName:  "top",
			wantIsDir: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got := dirEntryFromAttrs(tc.attrs, tc.prefix)

			if tc.wantNil {
				assert.Nil(t, got)
				return
			}

			require.NotNil(t, got)
			assert.Equal(t, tc.wantName, got.Name())
			assert.Equal(t, tc.wantIsDir, got.IsDir())

			if !tc.wantIsDir {
				info, err := got.Info()
				require.NoError(t, err)
				assert.Equal(t, tc.wantSize, info.Size())
			}
		})
	}
}
