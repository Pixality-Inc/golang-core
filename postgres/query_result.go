package postgres

type QueryResult interface {
	AffectedRows() int64
}

type QueryResultImpl struct {
	affectedRows int64
}

func NewEmptyQueryResult() QueryResult {
	return NewQueryResult(0)
}

func NewQueryResult(affectedRows int64) QueryResult {
	return &QueryResultImpl{
		affectedRows: affectedRows,
	}
}

func (q *QueryResultImpl) AffectedRows() int64 {
	return q.affectedRows
}
