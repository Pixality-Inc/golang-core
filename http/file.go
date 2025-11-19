package http

import "mime/multipart"

type File struct {
	File multipart.File
	Name string
	Size uint64
}

func NewFile(file multipart.File, name string, size uint64) *File {
	return &File{
		File: file,
		Name: name,
		Size: size,
	}
}
