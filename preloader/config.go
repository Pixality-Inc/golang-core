package cached_value

import "time"

type ConfigYaml struct {
	NameValue string        `env:"NAME" yaml:"name"`
	TTLValue  time.Duration `env:"TTL"  yaml:"ttl"`
}

func NewConfig(name string, ttl time.Duration) *ConfigYaml {
	return &ConfigYaml{
		NameValue: name,
		TTLValue:  ttl,
	}
}

func (c *ConfigYaml) Name() string {
	return c.NameValue
}

func (c *ConfigYaml) TTL() time.Duration {
	return c.TTLValue
}
