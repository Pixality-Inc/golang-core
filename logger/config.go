package logger

type Level string

const (
	TraceLevel Level = "trace"
	DebugLevel Level = "debug"
	InfoLevel  Level = "info"
	WarnLevel  Level = "warn"
	ErrorLevel Level = "error"
	FatalLevel Level = "fatal"
	PanicLevel Level = "panic"
)

type Format string

const (
	TextFormat Format = "text"
	JsonFormat Format = "json"
)

type Config interface {
	Level() Level
	Format() Format
	WithTimestamp() bool
	WithColors() bool
	WithStacktrace() bool
	WithStacktraceErrors() bool
}

var DefaultConfig = NewConfig(
	InfoLevel,
	TextFormat,
	true,
	true,
	false,
	false,
)

type ConfigImpl struct {
	level                Level
	format               Format
	withTimestamp        bool
	withColors           bool
	withStacktrace       bool
	withStacktraceErrors bool
}

func NewConfig(
	level Level,
	format Format,
	withTimestamp bool,
	withColors bool,
	withStacktrace bool,
	withStacktraceErrors bool,
) Config {
	return &ConfigImpl{
		level:                level,
		format:               format,
		withTimestamp:        withTimestamp,
		withColors:           withColors,
		withStacktrace:       withStacktrace,
		withStacktraceErrors: withStacktraceErrors,
	}
}

func (c *ConfigImpl) Level() Level {
	return c.level
}

func (c *ConfigImpl) Format() Format {
	return c.format
}

func (c *ConfigImpl) WithTimestamp() bool {
	return c.withTimestamp
}

func (c *ConfigImpl) WithColors() bool {
	return c.withColors
}

func (c *ConfigImpl) WithStacktrace() bool {
	return c.withStacktrace
}

func (c *ConfigImpl) WithStacktraceErrors() bool {
	return c.withStacktraceErrors
}
