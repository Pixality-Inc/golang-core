package postgres

// ScalarValue - wraps a single scalar result (e.g. SELECT COUNT(1) as value)
type ScalarValue[T any] struct {
	Value T `db:"value"`
}

type (
	IntValue     = ScalarValue[int]
	Int64Value   = ScalarValue[int64]
	Float64Value = ScalarValue[float64]
	BoolValue    = ScalarValue[bool]
)
