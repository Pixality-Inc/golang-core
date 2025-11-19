package json

type Entity[T any] struct {
	Entity T
	Data   RawMessage
}

func NewEntity[T any](object T) (*Entity[T], error) {
	entity := &Entity[T]{
		Entity: object,
		Data:   nil,
	}

	data, err := Marshal(object)
	if err != nil {
		return nil, err
	}

	entity.Data = data

	return entity, nil
}

func NewEntityFromByteArray[T any](data []byte) (*Entity[T], error) {
	entity := &Entity[T]{
		Data: nil,
	}

	if err := entity.UnmarshalJSON(data); err != nil {
		return nil, err
	}

	return entity, nil
}

func (e *Entity[T]) MarshalJSON() ([]byte, error) {
	return Marshal(e.Data)
}

func (e *Entity[T]) UnmarshalJSON(data []byte) error {
	e.Data = data

	if err := Unmarshal(data, &e.Entity); err != nil {
		return err
	}

	return nil
}
