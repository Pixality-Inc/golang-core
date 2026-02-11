package postgres

import "github.com/pixality-inc/squirrel"

func TableAlias(tableName string, alias string) string {
	return tableName + " AS " + alias
}

func Increment(column string, inc int) squirrel.Sqlizer {
	return squirrel.Expr(column+" + ?", inc)
}

func Decrement(column string, dec int) squirrel.Sqlizer {
	return squirrel.Expr(column+" - ?", dec)
}

func SortAsc(column string) string {
	return column + " ASC"
}

func SortDesc(column string) string {
	return column + " DESC"
}
