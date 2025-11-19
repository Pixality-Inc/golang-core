package util

func ApplyIfNotNil[E any, R any](value *E, apply func(*E) *R) *R {
	if IsNil(value) {
		return nil
	}

	return apply(value)
}

func ApplyIfNotNilDefault[E any, R any](value *E, defaultValue R, apply func(E) R) R {
	if IsNil(value) {
		return defaultValue
	}

	return apply(*value)
}
