package logger

type YamlConfig struct {
	LevelValue            Level  `env:"LEVEL"             yaml:"level"`
	FormatValue           Format `env:"FORMAT"            yaml:"format"`
	TimestampValue        bool   `env:"TIMESTAMP"         yaml:"timestamp"`
	ColorsValue           bool   `env:"COLORS"            yaml:"colors"`
	StacktraceValue       bool   `env:"STACKTRACE"        yaml:"stacktrace"`
	StacktraceErrorsValue bool   `env:"STACKTRACE_ERRORS" yaml:"stacktrace_errors"`
}

func (c *YamlConfig) Level() Level {
	return c.LevelValue
}

func (c *YamlConfig) Format() Format {
	return c.FormatValue
}

func (c *YamlConfig) WithTimestamp() bool {
	return c.TimestampValue
}

func (c *YamlConfig) WithColors() bool {
	return c.ColorsValue
}

func (c *YamlConfig) WithStacktrace() bool {
	return c.StacktraceValue
}

func (c *YamlConfig) WithStacktraceErrors() bool {
	return c.StacktraceErrorsValue
}
