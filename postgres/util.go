package postgres

import (
	"strings"

	"github.com/pixality-inc/squirrel"
)

func TableAlias(tableName string, alias string) string {
	return tableName + " AS " + alias
}

// All helpers double-quote identifiers following Postgres rules: an embedded
// double quote is escaped by doubling it (" -> ""), and NUL bytes (which Postgres
// forbids in identifiers) are dropped. Every other character — including
// backslashes and tabs — is literal inside a quoted identifier and passes through
// unchanged. This is the same scheme lib/pq.QuoteIdentifier and pgx use; note
// strconv/strings.Quote is NOT a substitute, as it applies Go escaping (\", \\,
// \t) which is invalid SQL.

func quoteIdent(col string) string {
	col = strings.ReplaceAll(col, "\x00", "")

	return `"` + strings.ReplaceAll(col, `"`, `""`) + `"`
}

func quoteIdents(columns []string) string {
	quoted := make([]string, len(columns))
	for i, col := range columns {
		quoted[i] = quoteIdent(col)
	}

	return strings.Join(quoted, ", ")
}

// OnConflict returns an ON CONFLICT clause with optional conflict target columns.
func OnConflict(columns ...string) string {
	if len(columns) == 0 {
		return "ON CONFLICT"
	}

	return "ON CONFLICT (" + quoteIdents(columns) + ")"
}

// OnConflictDoNothing returns an ON CONFLICT ... DO NOTHING clause.
func OnConflictDoNothing(columns ...string) string {
	return OnConflict(columns...) + " DO NOTHING"
}

// OnConflictDoUpdateSet returns an ON CONFLICT ... DO UPDATE SET clause with the given assignments.
// At least one assignment is required; zero assignments would produce invalid SQL.
func OnConflictDoUpdateSet(conflictColumns []string, assignments ...string) string {
	return OnConflict(conflictColumns...) + " DO UPDATE SET " + strings.Join(assignments, ", ")
}

// SetExcluded returns a "col = EXCLUDED.col" assignment expression.
func SetExcluded(col string) string {
	return quoteIdent(col) + " = EXCLUDED." + quoteIdent(col)
}

// SetNow returns a "col = NOW()" assignment expression.
func SetNow(col string) string {
	return quoteIdent(col) + " = NOW()"
}

// SetGreatestExcluded returns a "col = GREATEST(table.col, EXCLUDED.col)" assignment expression.
func SetGreatestExcluded(table, col string) string {
	return quoteIdent(col) + " = GREATEST(" + quoteIdent(table) + "." + quoteIdent(col) + ", EXCLUDED." + quoteIdent(col) + ")"
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
