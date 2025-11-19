package logger

type LogSpawner interface {
	NewLogger() Logger
}

type LogSpawnerImpl struct {
	log Logger
}

func NewLogSpawner(log Logger) LogSpawner {
	return &LogSpawnerImpl{
		log: log,
	}
}

func (s *LogSpawnerImpl) NewLogger() Logger {
	return s.log.Clone()
}

var LogSpawnerInstance = NewLogSpawner(NewDefault())

func InitLogSpawner(log Logger) error {
	LogSpawnerInstance = NewLogSpawner(log)

	return nil
}
