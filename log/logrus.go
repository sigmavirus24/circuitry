package log

import "github.com/sirupsen/logrus"

// Logrus implements circuitry's Logger interface using sirupsen/logrus
type Logrus struct {
	l logrus.FieldLogger
}

// Debug logs a message at the Debug level
func (l *Logrus) Debug(msg string) { l.l.Debug(msg) }

// Info logs a message at the Info level
func (l *Logrus) Info(msg string) { l.l.Info(msg) }

// Warn logs a message at the Warn level
func (l *Logrus) Warn(msg string) { l.l.Warn(msg) }

// Error logs a message at the Error level
func (l *Logrus) Error(msg string) { l.l.Error(msg) }

// WithError creates a new Logger with the context of the error included
func (l *Logrus) WithError(err error) Logger { return &Logrus{l.l.WithError(err)} }

// WithField creates a new Logger with the field and it's value included in
// the context
func (l *Logrus) WithField(field string, value any) Logger {
	return &Logrus{l.l.WithField(field, value)}
}

// WithFields creates a new Logger with the fields included in
// the context
func (l *Logrus) WithFields(fields Fields) Logger {
	return &Logrus{l.l.WithFields(logrus.Fields(fields))}
}

// NewLogrus creates a new Logrus implementation of the Logger interface using the
// specified *slog.Logger
func NewLogrus(logger logrus.FieldLogger) *Logrus {
	return &Logrus{logger}
}

var _ Logger = (*Logrus)(nil)
