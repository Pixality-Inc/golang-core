package logger

type Wrapper struct {
	log Logger
}

func NewWrapper(log Logger) *Wrapper {
	return &Wrapper{
		log: log,
	}
}

func (l *Wrapper) Debug(msg string, keyvals ...any) {
	entry, args := l.withArgs(msg, keyvals...)
	entry.Debug(args...)
}

func (l *Wrapper) Info(msg string, keyvals ...any) {
	entry, args := l.withArgs(msg, keyvals...)
	entry.Info(args...)
}

func (l *Wrapper) Warn(msg string, keyvals ...any) {
	entry, args := l.withArgs(msg, keyvals...)
	entry.Warn(args...)
}

func (l *Wrapper) Error(msg string, keyvals ...any) {
	entry, args := l.withArgs(msg, keyvals...)
	entry.Error(args...)
}

func (l *Wrapper) withSpaces(msg string, keyvals []any) []any {
	result := []any{msg}

	for _, v := range keyvals {
		result = append(result, " ")
		result = append(result, v)
	}

	return result
}

func (l *Wrapper) withArgs(msg string, keyvals ...any) (Logger, []any) {
	if len(keyvals)%2 == 0 {
		entry := l.log

		failed := false

		for i := 0; i < len(keyvals); i += 2 {
			switch v := keyvals[i].(type) {
			case string:
				entry = entry.WithField(v, keyvals[i+1])
			default:
				failed = true
			}
		}

		if failed {
			return l.log, l.withSpaces(msg, keyvals)
		} else {
			return entry, []any{msg}
		}
	} else {
		return l.log, l.withSpaces(msg, keyvals)
	}
}
