package storage

import (
	"mime"
	"path/filepath"
	"strings"
)

const ContentEncodingGzip = "gzip"

type Metadata interface {
	ContentType() string
	ContentEncoding() string
}

type MetadataImpl struct {
	contentType     string
	contentEncoding string
}

func NewMetadata(
	contentType string,
	contentEncoding string,
) *MetadataImpl {
	return &MetadataImpl{
		contentType:     contentType,
		contentEncoding: contentEncoding,
	}
}

func (m *MetadataImpl) ContentType() string {
	return m.contentType
}

func (m *MetadataImpl) ContentEncoding() string {
	return m.contentEncoding
}

func GetFileMetadataByName(filename string) (Metadata, error) {
	basename := filepath.Base(filename)
	ext1 := filepath.Ext(basename)

	var ext2 string

	name := strings.TrimSuffix(basename, ext1)
	if strings.Contains(name, ".") {
		ext2 = filepath.Ext(name)
	}

	if ext2 != "" {
		ext2, ext1 = ext1, ext2
	}

	contentType := mime.TypeByExtension(ext1)

	if strings.Contains(contentType, ";") {
		contentType = strings.TrimSpace(strings.Split(contentType, ";")[0])
	}

	var contentEncoding string

	if ext2 == ".gz" {
		contentEncoding = ContentEncodingGzip
	}

	metadata := NewMetadata(
		contentType,
		contentEncoding,
	)

	return metadata, nil
}
