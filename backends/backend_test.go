package backends_test

import (
	"context"
	"testing"
	"time"

	"github.com/sigmavirus24/circuitry"
	"github.com/sigmavirus24/circuitry/backends"
)

func TestInMemoryBackendLock(t *testing.T) {
	b := backends.NewInMemoryBackend()
	_, err := b.Lock(context.TODO(), "test")
	if err != nil {
		t.Fatalf("expected to get a lock but got err %+v", err)
	}
}

func TestInMemoryBackendRetrieve(t *testing.T) {
	b := backends.NewInMemoryBackend()
	ci, err := b.Retrieve(context.TODO(), "test")
	if err != nil {
		t.Fatalf("expected to get empty CircuitInformation but got err %+v", err)
	}
	if ci.State != circuitry.CircuitClosed {
		t.Fatalf("did not get default CircuitInformation")
	}
}

func TestInMemoryBackendStore(t *testing.T) {
	b := backends.NewInMemoryBackend()
	ci := circuitry.NewCircuitInformation(time.Hour * 2)
	ci.ConsecutiveSuccesses = 10
	ci.TotalSuccesses = 10
	ci.Total = 11
	ci.State = circuitry.CircuitHalfOpen
	err := b.Store(context.TODO(), "test", ci)
	if err != nil {
		t.Fatalf("expected no error but got %+v", err)
	}
	stored, err := b.Retrieve(context.TODO(), "test")
	if err != nil {
		t.Fatalf("expected no error retrieving CircuitInformation; got %+v", err)
	}
	if stored.Total != ci.Total || stored.State != ci.State || stored.TotalSuccesses != ci.TotalSuccesses || stored.ConsecutiveSuccesses != ci.ConsecutiveSuccesses {
		t.Fatalf("expected CircuitInformation to round-trip safely, but it didn't; expected: %+v, got %+v", ci, stored)
	}
}

func TestWithInMemoryBackend(t *testing.T) {
	s, err := circuitry.NewFactorySettings(backends.WithInMemoryBackend())
	if err != nil {
		t.Fatalf("expected NewFactorySettings to not error; got %v", err)
	}
	if s.StorageBackend == nil {
		t.Fatalf("expected WithInMemoryBackend to configure s.StorageBackend but it didn't")
	}
	_, ok := s.StorageBackend.(*backends.InMemoryBackend)
	if !ok {
		t.Fatalf("configured StorageBackend is not InMemoryBackend type; got %T", s.StorageBackend)
	}
}
