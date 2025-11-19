package util

func OrDefault[T any](ref *T, defaultValue T) T {
	if IsNil(ref) {
		return defaultValue
	} else {
		return *ref
	}
}
