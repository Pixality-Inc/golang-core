package cache

import "errors"

var (
	ErrMarshal   = errors.New("marshaling")
	ErrUnmarshal = errors.New("unmarshalling")
)

type Marshaller interface {
	Marshal(value any) ([]byte, error)
	Unmarshal(value []byte, result any) error
}
