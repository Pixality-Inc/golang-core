package flow

type Option = func(flowEngine *Impl)

func WithLogFiles(logsDir string, logFilePrefix string) Option {
	return func(flowEngine *Impl) {
		flowEngine.logsDir = &logsDir
		flowEngine.logFilePrefix = &logFilePrefix
	}
}
