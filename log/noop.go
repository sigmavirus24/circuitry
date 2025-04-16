package log

// NoOp implements a no-op version of the interface as a
// reasonable default for circuitry to use
type NoOp struct{}

// Debug will do nothing
func (l *NoOp) Debug(_ string) {}

// Info will do nothing
func (l *NoOp) Info(_ string) {}

// Warn will do nothing
func (l *NoOp) Warn(_ string) {}

// Error will do nothing
func (l *NoOp) Error(_ string) {}

// WithError will do nothing
func (l *NoOp) WithError(_ error) Logger { return l }

// WithField will do nothing
func (l *NoOp) WithField(_ string, _ any) Logger { return l }

// WithFields will do nothing
func (l *NoOp) WithFields(_ Fields) Logger { return l }

var _ Logger = (*NoOp)(nil)
