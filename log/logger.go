package log

// Logger provides an interface to be used so that logging can be provided by
// the circuitry package
type Logger interface {
	Debug(string)
	Info(string)
	Warn(string)
	Error(string)
	WithError(error) Logger
	WithField(string, any) Logger
	WithFields(Fields) Logger
}

// Fields that can be logged with a single call
type Fields map[string]any
