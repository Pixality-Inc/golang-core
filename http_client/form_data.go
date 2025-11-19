package http_client

import (
	"bytes"
	"io"
	"mime/multipart"
	"net/textproto"
)

type FormData struct {
	body   *bytes.Buffer
	writer *multipart.Writer
}

func NewFormData() *FormData {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	return &FormData{
		body:   body,
		writer: writer,
	}
}

func (f *FormData) AddField(name string, value string) error {
	return f.writer.WriteField(name, value)
}

func (f *FormData) AddFields(fields FormFields) error {
	for k, v := range fields {
		if err := f.writer.WriteField(k, v); err != nil {
			return err
		}
	}

	return nil
}

func (f *FormData) AddFile(fieldName string, fieldValue string, contentType string, body io.Reader) error {
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", multipart.FileContentDisposition(fieldName, fieldValue))
	h.Set("Content-Type", contentType)

	filePart, err := f.writer.CreatePart(h)
	if err != nil {
		return err
	}

	if _, err := io.Copy(filePart, body); err != nil {
		return err
	}

	return nil
}

func (f *FormData) Body() *bytes.Buffer {
	return f.body
}

func (f *FormData) ContentType() string {
	return f.writer.FormDataContentType()
}

func (f *FormData) Close() error {
	return f.writer.Close()
}
