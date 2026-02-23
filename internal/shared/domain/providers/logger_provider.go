package providers

type LoggerProvider interface {
	Info(msg string, fields ...any)
	Warn(msg string, fields ...any)
	Error(msg string, fields ...any)
	Debug(msg string, fields ...any)
	With(args ...any) LoggerProvider
	Sync() error
}
