package proto_parser

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
)

type Input interface {
	Name() string
	Source() (io.ReadCloser, error)
	Package() string
}

type FileInputImpl struct {
	filename string
	pkg      string
}

func NewFileInput(filename string, pkg string) Input {
	return &FileInputImpl{
		filename: filename,
		pkg:      pkg,
	}
}

func (i *FileInputImpl) Name() string {
	return filepath.Base(i.filename)
}

func (i *FileInputImpl) Source() (io.ReadCloser, error) {
	return os.Open(i.filename)
}

func (i *FileInputImpl) Package() string {
	return i.pkg
}

type BytesInputImpl struct {
	name   string
	source []byte
	pkg    string
}

func NewBytesInput(name string, source []byte, pkg string) Input {
	return &BytesInputImpl{
		name:   name,
		source: source,
		pkg:    pkg,
	}
}

func (i *BytesInputImpl) Name() string {
	return i.name
}

func (i *BytesInputImpl) Source() (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(i.source)), nil
}

func (i *BytesInputImpl) Package() string {
	return i.pkg
}
