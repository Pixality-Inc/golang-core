package json

type Object = map[string]any

func AsJsonObject(raw RawMessage) (Object, error) {
	var obj Object

	if err := Unmarshal(raw, &obj); err != nil {
		return nil, err
	}

	return obj, nil
}
