package postgres

// ValuePair - generic id+value result row for batch queries
type ValuePair[T comparable, V any] struct {
	Id    T `db:"id"`
	Value V `db:"value"`
}

// ValuePairArray - slice of ValuePair with lookup helpers
type ValuePairArray[T comparable, V any] []ValuePair[T, V]

func (a ValuePairArray[T, V]) Contains(id T) bool {
	for _, p := range a {
		if p.Id == id {
			return true
		}
	}

	return false
}

func (a ValuePairArray[T, V]) Value(id T) V {
	for _, p := range a {
		if p.Id == id {
			return p.Value
		}
	}

	var zero V

	return zero
}

// BoolPair - id+bool result row for batch queries
type BoolPair[T comparable] struct {
	Id T    `db:"id"`
	Ok bool `db:"ok"`
}

// BoolPairArray - slice of BoolPair with lookup helpers
type BoolPairArray[T comparable] []BoolPair[T]

func (a BoolPairArray[T]) Contains(id T) bool {
	for _, b := range a {
		if b.Id == id {
			return true
		}
	}

	return false
}

func (a BoolPairArray[T]) Value(id T) bool {
	for _, b := range a {
		if b.Id == id {
			return b.Ok
		}
	}

	return false
}

// IntPair - id+int result row for batch queries
type IntPair[T comparable] struct {
	Id    T   `db:"id"`
	Value int `db:"value"`
}

// IntPairArray - slice of IntPair with lookup helpers
type IntPairArray[T comparable] []IntPair[T]

func (a IntPairArray[T]) Contains(id T) bool {
	for _, p := range a {
		if p.Id == id {
			return true
		}
	}

	return false
}

func (a IntPairArray[T]) Value(id T) int {
	for _, p := range a {
		if p.Id == id {
			return p.Value
		}
	}

	return 0
}

// Float64Pair - id+float64 result row for batch queries
type Float64Pair[T comparable] struct {
	Id    T       `db:"id"`
	Value float64 `db:"value"`
}

// Float64PairArray - slice of Float64Pair with lookup helpers
type Float64PairArray[T comparable] []Float64Pair[T]

func (a Float64PairArray[T]) Contains(id T) bool {
	for _, p := range a {
		if p.Id == id {
			return true
		}
	}

	return false
}

func (a Float64PairArray[T]) Value(id T) float64 {
	for _, p := range a {
		if p.Id == id {
			return p.Value
		}
	}

	return 0
}
