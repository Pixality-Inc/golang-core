package logger

func FatalError(err error, args ...any) {
	LogSpawnerInstance.NewLogger().WithError(err).Fatal(args...)
}
