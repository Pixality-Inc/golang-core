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
