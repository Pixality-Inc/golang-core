package util

func MakeRef[T any](value T) *T {
	return &value
}
