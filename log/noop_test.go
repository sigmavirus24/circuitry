package log_test

import (
	"errors"
	"testing"

	"github.com/sigmavirus24/circuitry/log"
)

func TestNoOp(t *testing.T) {
	l := &log.NoOp{}
	l.WithError(errors.New("test noop err")).
		WithField("field", "value").
		WithFields(log.Fields{"a": "b", "c": "d"}).
		Debug("msg")
	l.Info("msg")
	l.Warn("msg")
	l.Error("msg")
}
