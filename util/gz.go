package util

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
)

func Gzip(data []byte) ([]byte, error) {
	var buffer bytes.Buffer

	gzWriter := gzip.NewWriter(&buffer)

	_, err := gzWriter.Write(data)
	if err != nil {
		return nil, fmt.Errorf("failed to compress data: %w", err)
	}

	if err = gzWriter.Flush(); err != nil {
		return nil, fmt.Errorf("failed to flush compresses data: %w", err)
	}

	if err = gzWriter.Close(); err != nil {
		return nil, fmt.Errorf("failed to close compresses data: %w", err)
	}

	return buffer.Bytes(), nil
}

func Gunzip(data []byte) ([]byte, error) {
	reader := bytes.NewReader(data)

	gz, err := gzip.NewReader(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}

	result, err := io.ReadAll(gz)
	if err != nil {
		return nil, fmt.Errorf("failed to read gzip data: %w", err)
	}

	return result, nil
}
