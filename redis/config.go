package redis

import "github.com/pixality-inc/golang-core/circuit_breaker"

type Config interface {
	SentinelMasterName() string
	SentinelAddresses() []string
	Network() string
	Protocol() int
	Host() string
	Port() int
	ClientName() string
	Username() string
	Password() string
	DB() int
	CircuitBreaker() circuit_breaker.Config
}

type ConfigYaml struct {
	SentinelMasterNameValue string                     `env:"SENTINEL_MASTER_NAME"    yaml:"sentinel_master_name"`
	SentinelAddressesValue  []string                   `env:"SENTINEL_ADDRESSES"      env-separator:","           yaml:"sentinel_addresses"`
	NetworkValue            string                     `env:"NETWORK"                 yaml:"network"`
	ProtocolValue           int                        `env:"PROTOCOL"                yaml:"protocol"`
	HostValue               string                     `env:"HOST"                    yaml:"host"`
	PortValue               int                        `env:"PORT"                    yaml:"port"`
	ClientNameValue         string                     `env:"CLIENT_NAME"             yaml:"client_name"`
	UsernameValue           string                     `env:"USERNAME"                yaml:"username"`
	PasswordValue           string                     `env:"PASSWORD"                yaml:"password"`
	DBValue                 int                        `env:"DB"                      yaml:"db"`
	CircuitBreakerValue     circuit_breaker.ConfigYaml `env-prefix:"CIRCUIT_BREAKER_" yaml:"circuit_breaker"`
}

func (c *ConfigYaml) SentinelMasterName() string {
	return c.SentinelMasterNameValue
}

func (c *ConfigYaml) SentinelAddresses() []string {
	return c.SentinelAddressesValue
}

func (c *ConfigYaml) Network() string {
	return c.NetworkValue
}

func (c *ConfigYaml) Protocol() int {
	return c.ProtocolValue
}

func (c *ConfigYaml) Host() string {
	return c.HostValue
}

func (c *ConfigYaml) Port() int {
	return c.PortValue
}

func (c *ConfigYaml) ClientName() string {
	return c.ClientNameValue
}

func (c *ConfigYaml) Username() string {
	return c.UsernameValue
}

func (c *ConfigYaml) Password() string {
	return c.PasswordValue
}

func (c *ConfigYaml) DB() int {
	return c.DBValue
}

func (c *ConfigYaml) CircuitBreaker() circuit_breaker.Config {
	return &c.CircuitBreakerValue
}
