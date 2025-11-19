package util

import (
	"errors"
	"fmt"

	"github.com/goccy/go-json"
	uuid "github.com/satori/go.uuid"
)

var (
	ErrUnmarshalByteArray = errors.New("unmarshal byte array error")
	ErrUnmarshalString    = errors.New("unmarshal string error")
	ErrUnmarshal          = errors.New("unmarshal error")
)

func UnmarshalJsonToId[T any](
	data []byte,
	defaultValue T,
	fn func(uuidValue uuid.UUID) T,
) (T, error) {
	var stringValue string

	err := json.Unmarshal(data, &stringValue)
	if err == nil {
		uuidValue, err := uuid.FromString(stringValue)
		if err != nil {
			return defaultValue, fmt.Errorf("%w: %s", errors.Join(ErrUnmarshalString, err), data)
		}

		return fn(uuidValue), nil
	}

	var byteArray []byte

	err = json.Unmarshal(data, &byteArray)
	if err == nil {
		uuidValue, err := uuid.FromBytes(byteArray)
		if err != nil {
			return defaultValue, fmt.Errorf("%w: %s", errors.Join(ErrUnmarshalByteArray, err), data)
		}

		return fn(uuidValue), nil
	}

	return defaultValue, fmt.Errorf("%w: %s", ErrUnmarshal, data)
}
