package http

import (
	"net"
	"strconv"
	"time"
)

type Config interface {
	Address() string
	ShutdownTimeout() time.Duration
}

type ConfigYaml struct {
	HostValue            string        `env:"HOST"             yaml:"host"`
	PortValue            int           `env:"PORT"             yaml:"port"`
	ShutdownTimeoutValue time.Duration `env:"SHUTDOWN_TIMEOUT" yaml:"shutdown_timeout"`
}

func (c *ConfigYaml) Address() string {
	return net.JoinHostPort(c.HostValue, strconv.Itoa(c.PortValue))
}

func (c *ConfigYaml) ShutdownTimeout() time.Duration {
	return c.ShutdownTimeoutValue
}
