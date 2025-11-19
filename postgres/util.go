package postgres

func TableAlias(tableName string, alias string) string {
	return tableName + " AS " + alias
}
