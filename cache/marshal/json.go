package marshal

import (
	"github.com/pixality-inc/golang-core/cache"
	"github.com/pixality-inc/golang-core/json"
)

type JsonMarshaller struct {
	cache.Marshaller
}

func NewJsonMarshaller() *JsonMarshaller {
	return &JsonMarshaller{}
}

func (m *JsonMarshaller) Marshal(value any) ([]byte, error) {
	return json.Marshal(value)
}

func (m *JsonMarshaller) Unmarshal(value []byte, result any) error {
	if err := json.Unmarshal(value, &result); err != nil {
		return err
	}

	return nil
}
