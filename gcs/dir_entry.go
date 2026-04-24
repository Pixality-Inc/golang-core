package gcs

import (
	"io/fs"
	"time"
)

// dirEntry implements storage.DirEntry and fs.FileInfo for GCS list results.
// For CommonPrefixes (dirs) size and modTime stay zero since GCS does not
// expose them.
type dirEntry struct {
	name    string
	isDir   bool
	size    int64
	modTime time.Time
}

func (e *dirEntry) Name() string       { return e.name }
func (e *dirEntry) IsDir() bool        { return e.isDir }
func (e *dirEntry) Size() int64        { return e.size }
func (e *dirEntry) ModTime() time.Time { return e.modTime }
func (e *dirEntry) Sys() any           { return nil }

func (e *dirEntry) Mode() fs.FileMode {
	if e.isDir {
		return fs.ModeDir
	}

	return 0
}

func (e *dirEntry) Type() fs.FileMode {
	return e.Mode()
}

func (e *dirEntry) Info() (fs.FileInfo, error) {
	return e, nil
}
