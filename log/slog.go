//go:build go1.21

// +bulid go1.21

package log

import "log/slog"

// SLog wraps the standard library log/slog logger for use within circuitry
type SLog struct {
	l *slog.Logger
}

// Debug logs a message at the Debug level
func (l *SLog) Debug(msg string) { l.l.Debug(msg) }

// Info logs a message at the Info level
func (l *SLog) Info(msg string) { l.l.Info(msg) }

// Warn logs a message at the Warn level
func (l *SLog) Warn(msg string) { l.l.Warn(msg) }

// Error logs a message at the Error level
func (l *SLog) Error(msg string) { l.l.Error(msg) }

// WithError creates a new Logger with the context of the error included
func (l *SLog) WithError(err error) Logger { return &SLog{l.l.With("err", err)} }

// WithField creates a new Logger with the field and it's value included in
// the context
func (l *SLog) WithField(field string, value any) Logger { return &SLog{l.l.With(field, value)} }

// WithFields creates a new Logger with the fields included in
// the context
func (l *SLog) WithFields(fields Fields) Logger {
	values := make([]any, 0, 2*len(fields))
	for key, value := range fields {
		values = append(values, key, value)
	}
	return &SLog{l.l.With(values...)}
}

// NewSLog creates a new SLog implementation of the Logger interface using the
// specified *slog.Logger
func NewSLog(logger *slog.Logger) *SLog {
	return &SLog{logger}
}

var _ Logger = (*SLog)(nil)
