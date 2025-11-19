package postgres

import (
	"fmt"
	"net"
	"strconv"

	"github.com/pixality-inc/golang-core/circuit_breaker"
)

type DatabaseConfig interface {
	Name() string
	PoolMax() int
	CircuitBreaker() circuit_breaker.Config
	DSN() string
	ParamsUrl() string
}

type DatabaseConfigYaml struct {
	NameValue              string                     `env:"NAME"                    yaml:"name"`
	HostValue              string                     `env:"HOST"                    yaml:"host"`
	PortValue              int                        `env:"PORT"                    yaml:"port"`
	UserValue              string                     `env:"USER"                    yaml:"user"`
	PasswordValue          string                     `env:"PASSWORD"                yaml:"password"`
	DatabaseValue          string                     `env:"DATABASE"                yaml:"database"`
	SchemaValue            string                     `env:"SCHEMA"                  yaml:"schema"`
	PoolMaxValue           int                        `env:"POOL_MAX"                yaml:"pool_max"`
	AppNameValue           string                     `env:"APP_NAME"                yaml:"app_name"`
	ConnectionTimeoutValue int                        `env:"CONNECTION_TIMEOUT"      yaml:"connection_timeout"`
	CircuitBreakerValue    circuit_breaker.ConfigYaml `env-prefix:"CIRCUIT_BREAKER_" yaml:"circuit_breaker"`
}

func (c *DatabaseConfigYaml) Name() string {
	return c.NameValue
}

func (c *DatabaseConfigYaml) PoolMax() int {
	return c.PoolMaxValue
}

func (c *DatabaseConfigYaml) CircuitBreaker() circuit_breaker.Config {
	return &c.CircuitBreakerValue
}

func (c *DatabaseConfigYaml) DSN() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s/%s?application_name=%s&search_path=%s&connect_timeout=%d",
		c.UserValue,
		c.PasswordValue,
		net.JoinHostPort(c.HostValue, strconv.Itoa(c.PortValue)),
		c.DatabaseValue,
		c.AppNameValue,
		c.SchemaValue,
		c.ConnectionTimeoutValue,
	)
}

func (c *DatabaseConfigYaml) ParamsUrl() string {
	return fmt.Sprintf(
		"host=%s port=%d dbname=%s user=%s password=%s application_name=%s search_path=%s sslmode=disable",
		c.HostValue,
		c.PortValue,
		c.DatabaseValue,
		c.UserValue,
		c.PasswordValue,
		c.AppNameValue,
		c.SchemaValue,
	)
}
