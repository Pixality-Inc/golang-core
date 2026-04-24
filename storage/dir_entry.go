package storage

import (
	"io/fs"
	"time"
)

// dirEntry is a value-only snapshot that implements DirEntry and fs.FileInfo.
// It is meant for providers whose underlying listing API does not return an
// fs.DirEntry natively (e.g. S3, GCS). Use NewFileEntry / NewDirEntry to
// construct instances.
type dirEntry struct {
	name    string
	isDir   bool
	size    int64
	modTime time.Time
}

// NewFileEntry builds a DirEntry representing a regular file.
func NewFileEntry(name string, size int64, modTime time.Time) DirEntry {
	return &dirEntry{
		name:    name,
		size:    size,
		modTime: modTime,
	}
}

// NewDirEntry builds a DirEntry representing a directory. Remote object
// storage typically does not expose a size or mod time for directories, so
// both stay zero.
func NewDirEntry(name string) DirEntry {
	return &dirEntry{
		name:  name,
		isDir: true,
	}
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
