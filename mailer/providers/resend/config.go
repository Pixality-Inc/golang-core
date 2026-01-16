package resend

type ConfigYaml interface {
	ApiKey() string
}

type ConfigYamlImpl struct {
	ApiKeyValue string `env:"API_KEY" yaml:"api_key"`
}

func (c *ConfigYamlImpl) ApiKey() string {
	return c.ApiKeyValue
}
