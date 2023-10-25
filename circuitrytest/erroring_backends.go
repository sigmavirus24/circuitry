package circuitrytest

import (
	"context"
	"sync"

	"github.com/sigmavirus24/circuitry"
)

// ErroringInMemoryBackend provides a backend that always errors. Useful for
// testing circuitry and tools using it
type ErroringInMemoryBackend struct {
	RetrieveError error
	StoreError    error
	LockError     error
}

// Store implements the StorageBackender interface but always returns the
// configured StoreError
func (b ErroringInMemoryBackend) Store(_ context.Context, _ string, _ circuitry.CircuitInformation) error {
	return b.StoreError
}

// Retrieve implements the StorageBackender interface but always returns the
// configured RetrieveError and an empty CircuitInformation
func (b ErroringInMemoryBackend) Retrieve(_ context.Context, _ string) (circuitry.CircuitInformation, error) {
	return circuitry.CircuitInformation{}, b.RetrieveError
}

// Lock implements the StorageBackender interface. It returns a mutex if
// LockError == nil, otherwise it always returns the configured LockError
func (b ErroringInMemoryBackend) Lock(_ context.Context, _ string) (sync.Locker, error) {
	if b.LockError == nil {
		return &sync.Mutex{}, nil
	}
	return nil, b.LockError
}

var _ circuitry.StorageBackender = (*ErroringInMemoryBackend)(nil)
