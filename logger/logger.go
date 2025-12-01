package logger

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/pkgerrors"
)

type Fields = map[string]any

type ClonableLogger interface {
	Clone() Logger
}

type OutputLogger interface {
	WithOutput(writer io.Writer) Logger
	WithStdoutOutput(writer io.Writer) Logger
	WithStderrOutput(writer io.Writer) Logger
}

type FieldsLogger interface {
	WithField(key string, value any) Logger
	WithFields(fields Fields) Logger
	WithError(err error) Logger
}

type FormatLogger interface {
	Tracef(format string, args ...any)
	Debugf(format string, args ...any)
	Infof(format string, args ...any)
	Warnf(format string, args ...any)
	Errorf(format string, args ...any)
	Fatalf(format string, args ...any)
	Panicf(format string, args ...any)
}

type ArgsLogger interface {
	Trace(args ...any)
	Debug(args ...any)
	Info(args ...any)
	Warn(args ...any)
	Error(args ...any)
	Fatal(args ...any)
	Panic(args ...any)
}

type NewLineLogger interface {
	Traceln(args ...any)
	Debugln(args ...any)
	Infoln(args ...any)
	Warnln(args ...any)
	Errorln(args ...any)
	Fatalln(args ...any)
	Panicln(args ...any)
}

type Logger interface {
	ClonableLogger
	OutputLogger
	FieldsLogger
	FormatLogger
	ArgsLogger
}

type internalConfig struct {
	isTrace              bool
	withStacktraceErrors bool
}

type Impl struct {
	stdoutLogger zerolog.Logger
	stderrLogger zerolog.Logger
	config       *internalConfig
}

func New(cfg Config) Logger {
	zerologLevel := convertLevel(cfg.Level())

	stdoutLogger := createLogger(
		os.Stdout,
		zerologLevel,
		cfg.Format(),
		cfg.WithTimestamp(),
		cfg.WithColors(),
		cfg.WithStacktrace(),
	)

	stderrLogger := createLogger(
		os.Stderr,
		zerologLevel,
		cfg.Format(),
		cfg.WithTimestamp(),
		cfg.WithColors(),
		cfg.WithStacktrace(),
	)

	config := &internalConfig{
		isTrace:              cfg.Level() == TraceLevel,
		withStacktraceErrors: cfg.WithStacktraceErrors(),
	}

	return newWithLoggers(stdoutLogger, stderrLogger, config)
}

func NewDefault() Logger {
	return New(DefaultConfig)
}

func (l *Impl) Clone() Logger {
	return newWithLoggers(
		l.stdoutLogger.With().Logger(),
		l.stderrLogger.With().Logger(),
		l.config,
	)
}

func (l *Impl) WithOutput(writer io.Writer) Logger {
	return l.WithStdoutOutput(writer).WithStderrOutput(writer)
}

func (l *Impl) WithStdoutOutput(writer io.Writer) Logger {
	return newWithLoggers(
		l.stdoutLogger.Output(writer),
		l.stderrLogger,
		l.config,
	)
}

func (l *Impl) WithStderrOutput(writer io.Writer) Logger {
	return newWithLoggers(
		l.stdoutLogger,
		l.stderrLogger.Output(writer),
		l.config,
	)
}

func (l *Impl) WithField(key string, value any) Logger {
	return l.WithFields(Fields{
		key: value,
	})
}

func (l *Impl) WithFields(fields Fields) Logger {
	modifyLogger := func(logger zerolog.Logger) zerolog.Logger {
		if len(fields) > 0 {
			return logger.With().Fields(fields).Logger()
		} else {
			return logger
		}
	}

	return newWithLoggers(
		modifyLogger(l.stdoutLogger),
		modifyLogger(l.stderrLogger),
		l.config,
	)
}

func (l *Impl) WithError(err error) Logger {
	modifyLogger := func(logger zerolog.Logger) zerolog.Logger {
		if l.config.withStacktraceErrors {
			return logger.With().Stack().Err(err).Logger()
		}

		return logger.With().Err(err).Logger()
	}

	return newWithLoggers(
		modifyLogger(l.stdoutLogger),
		modifyLogger(l.stderrLogger),
		l.config,
	)
}

func (l *Impl) Tracef(format string, args ...any) {
	l.Trace(fmt.Sprintf(format, args...))
}

func (l *Impl) Debugf(format string, args ...any) {
	l.Debug(fmt.Sprintf(format, args...))
}

func (l *Impl) Infof(format string, args ...any) {
	l.Info(fmt.Sprintf(format, args...))
}

func (l *Impl) Warnf(format string, args ...any) {
	l.Warn(fmt.Sprintf(format, args...))
}

func (l *Impl) Errorf(format string, args ...any) {
	l.Error(fmt.Sprintf(format, args...))
}

func (l *Impl) Fatalf(format string, args ...any) {
	l.Fatal(fmt.Sprintf(format, args...))
}

func (l *Impl) Panicf(format string, args ...any) {
	l.Panic(fmt.Sprintf(format, args...))
}

func (l *Impl) Trace(args ...any) {
	if l.config.isTrace {
		l.stdoutLogger.Debug().Msg(fmt.Sprint(args...))
	}
}

func (l *Impl) Debug(args ...any) {
	l.stdoutLogger.Debug().Msg(fmt.Sprint(args...))
}

func (l *Impl) Info(args ...any) {
	l.stdoutLogger.Info().Msg(fmt.Sprint(args...))
}

func (l *Impl) Warn(args ...any) {
	l.stderrLogger.Warn().Msg(fmt.Sprint(args...))
}

func (l *Impl) Error(args ...any) {
	l.stderrLogger.Error().Msg(fmt.Sprint(args...))
}

func (l *Impl) Fatal(args ...any) {
	l.stderrLogger.Fatal().Msg(fmt.Sprint(args...))
}

func (l *Impl) Panic(args ...any) {
	l.stderrLogger.Panic().Msg(fmt.Sprint(args...))
}

func newWithLoggers(
	stdoutLogger zerolog.Logger,
	stderrLogger zerolog.Logger,
	config *internalConfig,
) *Impl {
	return &Impl{
		stdoutLogger: stdoutLogger,
		stderrLogger: stderrLogger,
		config:       config,
	}
}

func convertLevel(level Level) zerolog.Level {
	switch level {
	case TraceLevel:
		return zerolog.DebugLevel
	case DebugLevel:
		return zerolog.DebugLevel
	case InfoLevel:
		return zerolog.InfoLevel
	case WarnLevel:
		return zerolog.WarnLevel
	case ErrorLevel:
		return zerolog.ErrorLevel
	case FatalLevel:
		return zerolog.FatalLevel
	case PanicLevel:
		return zerolog.PanicLevel
	default:
		return zerolog.InfoLevel
	}
}

func createLogger(
	out io.Writer,
	level zerolog.Level,
	format Format,
	withTimestamp bool,
	withColors bool,
	withStacktrace bool,
) zerolog.Logger {
	logger := zerolog.New(out)

	switch format {
	case JsonFormat:
	case TextFormat, NoFormat:
		logger = logger.Output(zerolog.ConsoleWriter{
			Out:        out,
			NoColor:    !withColors,
			TimeFormat: time.RFC3339,
		})
	}

	with := logger.Level(level).With()

	if withTimestamp {
		with = with.Timestamp()
	}

	if withStacktrace {
		with = with.Stack()
	}

	return with.Logger()
}

// nolint:gochecknoinits
func init() {
	// nolint:reassign
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack
}
