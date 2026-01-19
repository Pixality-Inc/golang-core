package smtp

type ConfigYaml interface {
	Host() string
	Port() int
	Username() string
	Password() string
}

type ConfigYamlImpl struct {
	HostValue     string `env:"HOST"     yaml:"host"`
	PortValue     int    `env:"PORT"     yaml:"port"`
	UsernameValue string `env:"USERNAME" yaml:"username"`
	PasswordValue string `env:"PASSWORD" yaml:"password"`
}

func (c *ConfigYamlImpl) Host() string {
	return c.HostValue
}

func (c *ConfigYamlImpl) Port() int {
	return c.PortValue
}

func (c *ConfigYamlImpl) Username() string {
	return c.UsernameValue
}

func (c *ConfigYamlImpl) Password() string {
	return c.PasswordValue
}
