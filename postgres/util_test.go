package postgres_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pixality-inc/golang-core/postgres"
)

func TestTableAlias(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "users AS u", postgres.TableAlias("users", "u"))
}

func TestIncrement(t *testing.T) {
	t.Parallel()

	sql, args, err := postgres.Increment("counter", 5).ToSql()
	require.NoError(t, err)
	assert.Equal(t, "counter + ?", sql)
	assert.Equal(t, []any{5}, args)
}

func TestDecrement(t *testing.T) {
	t.Parallel()

	sql, args, err := postgres.Decrement("counter", 3).ToSql()
	require.NoError(t, err)
	assert.Equal(t, "counter - ?", sql)
	assert.Equal(t, []any{3}, args)
}

func TestSort(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "created_at ASC", postgres.SortAsc("created_at"))
	assert.Equal(t, "created_at DESC", postgres.SortDesc("created_at"))
}

func TestOnConflict(t *testing.T) {
	t.Parallel()

	assert.Equal(t, `ON CONFLICT`, postgres.OnConflict())
	assert.Equal(t, `ON CONFLICT ("id")`, postgres.OnConflict("id"))
	assert.Equal(t, `ON CONFLICT ("org_id", "key")`, postgres.OnConflict("org_id", "key"))
}

func TestQuoteIdentEscaping(t *testing.T) {
	t.Parallel()

	// embedded double quote is doubled, not Go-escaped
	assert.Equal(t, `ON CONFLICT ("a""b")`, postgres.OnConflict(`a"b`))
	// backslash stays literal (strings.Quote would emit \\)
	assert.Equal(t, "ON CONFLICT (\"a\\b\")", postgres.OnConflict(`a\b`))
	// tab stays literal (strings.Quote would emit \t)
	assert.Equal(t, "ON CONFLICT (\"a\tb\")", postgres.OnConflict("a\tb"))
	// NUL bytes are forbidden in Postgres identifiers and get dropped
	assert.Equal(t, `ON CONFLICT ("ab")`, postgres.OnConflict("a\x00b"))
}

func TestOnConflictDoNothing(t *testing.T) {
	t.Parallel()

	assert.Equal(t, `ON CONFLICT DO NOTHING`, postgres.OnConflictDoNothing())
	assert.Equal(t, `ON CONFLICT ("id") DO NOTHING`, postgres.OnConflictDoNothing("id"))
	assert.Equal(t,
		`ON CONFLICT ("location_id", "user_id") DO NOTHING`,
		postgres.OnConflictDoNothing("location_id", "user_id"),
	)
}

func TestOnConflictDoUpdateSet(t *testing.T) {
	t.Parallel()

	assert.Equal(t,
		`ON CONFLICT ("org_id", "provider") DO UPDATE SET "data" = EXCLUDED."data", "updated_at" = EXCLUDED."updated_at"`,
		postgres.OnConflictDoUpdateSet(
			[]string{"org_id", "provider"},
			postgres.SetExcluded("data"),
			postgres.SetExcluded("updated_at"),
		),
	)
}

func TestSetExcluded(t *testing.T) {
	t.Parallel()

	assert.Equal(t, `"etag" = EXCLUDED."etag"`, postgres.SetExcluded("etag"))
}

func TestSetNow(t *testing.T) {
	t.Parallel()

	assert.Equal(t, `"updated_at" = NOW()`, postgres.SetNow("updated_at"))
}

func TestSetGreatestExcluded(t *testing.T) {
	t.Parallel()

	assert.Equal(t,
		`"read_until_at" = GREATEST("assets_read_cursors"."read_until_at", EXCLUDED."read_until_at")`,
		postgres.SetGreatestExcluded("assets_read_cursors", "read_until_at"),
	)
}
