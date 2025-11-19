package postgres

type PlainSql struct {
	sql    string
	params []any
}

func NewPlainSql(sql string, params ...any) *PlainSql {
	return &PlainSql{
		sql:    sql,
		params: params,
	}
}

func (p PlainSql) ToSql() (string, []any, error) {
	return p.sql, p.params, nil
}
