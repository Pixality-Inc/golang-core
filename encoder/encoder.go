package encoder

import (
	"encoding/base64"

	"github.com/pixality-inc/golang-core/errors"
)

var ErrBase64Decode = errors.New("encoder.base64_decode", "base64 decode error")

type Encoder interface {
	Encode(data []byte) []byte
	Decode(data []byte) ([]byte, error)
	EncodeString(data string) string
	DecodeString(data string) ([]byte, error)
}

type Impl struct {
	key []byte
}

func New(key []byte) *Impl {
	return &Impl{
		key: key,
	}
}

func (e *Impl) Encode(data []byte) []byte {
	return xorEncrypt(e.key, data)
}

func (e *Impl) Decode(data []byte) ([]byte, error) {
	return xorDecrypt(e.key, data), nil
}

func (e *Impl) EncodeString(data string) string {
	encryptedData := e.Encode([]byte(data))

	return base64.StdEncoding.EncodeToString(encryptedData)
}

func (e *Impl) DecodeString(data string) (string, error) {
	dataDecrypted, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return "", errors.Join(ErrBase64Decode, err)
	}

	decoded, err := e.Decode(dataDecrypted)
	if err != nil {
		return "", err
	}

	return string(decoded), nil
}

func xorEncrypt(key, data []byte) []byte {
	res := make([]byte, len(data))

	for index, s := range data {
		res[index] = s ^ key[(index)%len(key)]
	}

	return res
}

func xorDecrypt(key []byte, data []byte) (result []byte) {
	res := make([]byte, len(data))

	for index, s := range data {
		res[index] = s ^ key[(index)%len(key)]
	}

	return res
}
