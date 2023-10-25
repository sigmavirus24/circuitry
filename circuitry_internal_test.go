package circuitry

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/sigmavirus24/circuitry/log"
)

func newFactory(opts ...SettingsOption) *CircuitBreakerFactory {
	settings, _ := NewFactorySettings(opts...)
	return NewCircuitBreakerFactory(settings)
}

type NoOpBackend struct{}

func (b *NoOpBackend) Store(_ context.Context, _ string, _ CircuitInformation) error { return nil }
func (b *NoOpBackend) Retrieve(_ context.Context, _ string) (CircuitInformation, error) {
	return CircuitInformation{}, nil
}
func (b *NoOpBackend) Lock(_ context.Context, _ string) (sync.Locker, error) {
	return &sync.Mutex{}, nil
}

func TestSetState(t *testing.T) {
	factory := newFactory(
		WithStorageBackend(&NoOpBackend{}),
		WithFailureCountThreshold(1),
		WithCloseThreshold(2),
		WithDefaultTripFunc(),
		WithAllowAfter(5*time.Millisecond),
	)
	breaker := factory.BreakerFor("TestSetState", map[string]any{})
	cb, _ := breaker.(*circuitBreaker)
	if cb.state != CircuitClosed {
		t.Fatalf("expected cb.state = %s; got %s", CircuitClosed, cb.state)
	}
	cb.setState(CircuitClosed, time.Now())
	if cb.state != CircuitClosed {
		t.Fatalf("expected cb.state = %s; got %s", CircuitClosed, cb.state)
	}
}

func TestDefaultNoOpLogger(t *testing.T) {
	s, err := NewFactorySettings()
	if err != nil {
		t.Fatalf("expected to not receive an error but got %v", err)
	}
	factory := NewCircuitBreakerFactory(s)
	breaker := factory.BreakerFor("breaker", map[string]any{})
	actualBreaker, _ := breaker.(*circuitBreaker)
	if _, ok := actualBreaker.logger.(*log.NoOp); !ok {
		t.Fatalf("expected logger to default to log.NoOp; got %T", actualBreaker.logger)
	}
}
