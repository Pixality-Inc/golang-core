package util

import "sort"

func Map[IN any, OUT any](slice []IN, mappingFunction func(x IN) (OUT, error)) ([]OUT, error) {
	if len(slice) == 0 {
		return nil, nil
	}

	out := make([]OUT, 0, len(slice))

	for _, v := range slice {
		result, err := mappingFunction(v)
		if err != nil {
			return nil, err
		}

		out = append(out, result)
	}

	return out, nil
}

func MapSimple[IN any, OUT any](slice []IN, mappingFunction func(x IN) OUT) []OUT {
	if len(slice) == 0 {
		return nil
	}

	out := make([]OUT, 0, len(slice))

	for _, v := range slice {
		out = append(out, mappingFunction(v))
	}

	return out
}

func SliceUnique[T comparable](slice []T) []T {
	if len(slice) == 0 {
		return nil
	}

	sliceMap := make(map[T]bool)

	for _, v := range slice {
		sliceMap[v] = true
	}

	results := make([]T, 0, len(sliceMap))

	for k := range sliceMap {
		results = append(results, k)
	}

	return results
}

func SliceSort[T any](slice []T, comp func(T, T) bool) {
	sort.Slice(slice, func(i, j int) bool {
		return comp(slice[i], slice[j])
	})
}

func SliceSum[T any, R any](slice []T, start R, reducer func(prev R, current T) R) R {
	result := start

	for _, item := range slice {
		result = reducer(result, item)
	}

	return result
}
