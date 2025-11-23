package http_client

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/textproto"
)

type FormDataInterface interface {
	AddField(name string, value string) error
	AddFields(fields FormFields) error
	AddFile(fieldName string, fileName string, contentType string, body io.Reader) error
}

type formFile struct {
	fieldName   string
	fileName    string
	contentType string
	content     []byte
}

type FormData struct {
	fields            map[string]string
	files             []formFile
	cachedBody        *bytes.Buffer
	cachedContentType string
	built             bool
}

func NewFormData() *FormData {
	return &FormData{
		fields: make(map[string]string),
		files:  make([]formFile, 0),
	}
}

func (f *FormData) AddField(name string, value string) error {
	f.fields[name] = value
	return nil
}

func (f *FormData) AddFields(fields FormFields) error {
	for k, v := range fields {
		f.fields[k] = v
	}
	return nil
}

func (f *FormData) AddFile(fieldName string, fileName string, contentType string, body io.Reader) error {
	content, err := io.ReadAll(body)
	if err != nil {
		return err
	}

	f.files = append(f.files, formFile{
		fieldName:   fieldName,
		fileName:    fileName,
		contentType: contentType,
		content:     content,
	})

	return nil
}

func (f *FormData) build() (*bytes.Buffer, string, error) {
	if f.built {
		return f.cachedBody, f.cachedContentType, nil
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	for name, value := range f.fields {
		if err := writer.WriteField(name, value); err != nil {
			return nil, "", err
		}
	}

	for _, file := range f.files {
		h := make(textproto.MIMEHeader)
		h.Set("Content-Disposition", multipart.FileContentDisposition(file.fieldName, file.fileName))
		h.Set("Content-Type", file.contentType)

		part, err := writer.CreatePart(h)
		if err != nil {
			return nil, "", err
		}

		if _, err := part.Write(file.content); err != nil {
			return nil, "", err
		}
	}

	if err := writer.Close(); err != nil {
		return nil, "", err
	}

	f.cachedBody = body
	f.cachedContentType = writer.FormDataContentType()
	f.built = true

	return f.cachedBody, f.cachedContentType, nil
}

func (f *FormData) Build() (*bytes.Buffer, string, error) {
	return f.build()
}
