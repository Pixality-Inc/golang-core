package storage

import (
	"io/fs"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFileEntry(t *testing.T) {
	t.Parallel()

	now := time.Unix(1700000000, 0).UTC()
	entry := NewFileEntry("video.mp4", 1024, now)

	assert.Equal(t, "video.mp4", entry.Name())
	assert.False(t, entry.IsDir())
	assert.Equal(t, fs.FileMode(0), entry.Type())

	info, err := entry.Info()
	require.NoError(t, err)
	assert.Equal(t, "video.mp4", info.Name())
	assert.Equal(t, int64(1024), info.Size())
	assert.Equal(t, now, info.ModTime())
	assert.Equal(t, fs.FileMode(0), info.Mode())
	assert.False(t, info.IsDir())
	assert.Nil(t, info.Sys())
}

func TestNewDirEntry(t *testing.T) {
	t.Parallel()

	entry := NewDirEntry("subdir")

	assert.Equal(t, "subdir", entry.Name())
	assert.True(t, entry.IsDir())
	assert.Equal(t, fs.ModeDir, entry.Type())

	info, err := entry.Info()
	require.NoError(t, err)
	assert.Equal(t, "subdir", info.Name())
	assert.Equal(t, int64(0), info.Size())
	assert.True(t, info.ModTime().IsZero())
	assert.Equal(t, fs.ModeDir, info.Mode())
	assert.True(t, info.IsDir())
}
