package postgres_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/pixality-inc/golang-core/postgres"
)

func newDatabaseConfig() *postgres.DatabaseConfigYaml {
	return &postgres.DatabaseConfigYaml{
		NameValue:              "primary",
		HostValue:              "db.example.com",
		PortValue:              5432,
		UserValue:              "app_user",
		PasswordValue:          "secret",
		DatabaseValue:          "app_db",
		SchemaValue:            "public",
		PoolMaxValue:           10,
		AppNameValue:           "my_app",
		ConnectionTimeoutValue: 5,
	}
}

func TestDatabaseConfigGetters(t *testing.T) {
	t.Parallel()

	config := newDatabaseConfig()

	assert.Equal(t, "primary", config.Name())
	assert.Equal(t, 10, config.PoolMax())
	require.NotNil(t, config.CircuitBreaker())
}

func TestDatabaseConfigDSN(t *testing.T) {
	t.Parallel()

	expected := "postgres://app_user:secret@db.example.com:5432/app_db" +
		"?application_name=my_app&search_path=public&connect_timeout=5"

	assert.Equal(t, expected, newDatabaseConfig().DSN())
}

func TestDatabaseConfigParamsUrl(t *testing.T) {
	t.Parallel()

	expected := "host=db.example.com port=5432 dbname=app_db user=app_user" +
		" password=secret application_name=my_app search_path=public sslmode=disable"

	assert.Equal(t, expected, newDatabaseConfig().ParamsUrl())
}
