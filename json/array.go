package json

type Array = []any

func NewArray(elements ...any) Array {
	return elements
}

func AsJsonArray[T any](array []T) Array {
	arr := make(Array, 0, len(array))

	for _, element := range array {
		arr = append(arr, element)
	}

	return arr
}
